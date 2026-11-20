package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"
)

const (
	githubWebURL = "https://github.com"
	githubAPIURL = "https://api.github.com"
)

type createdGitHubApp struct {
	ID      int64  `json:"id"`
	Slug    string `json:"slug"`
	PEM     string `json:"pem"`
	HTMLURL string `json:"html_url"`
}

type githubAppRequest struct {
	name  string
	owner string // an organization; empty for the user's own account
	web   string
	api   string
}

var appFormPage = template.Must(template.New("form").Parse(`<!doctype html>
<meta charset="utf-8"><title>Create the GitHub App</title>
<form id="f" method="post" action="{{.Action}}">
<input type="hidden" name="manifest" value="{{.Manifest}}">
<button>Continue to GitHub</button>
</form>
<script>document.getElementById("f").submit()</script>`))

// createGitHubApp uses GitHub's App manifest flow: the person only presses
// Create on GitHub, and the App's ID and private key come back to this
// process. The alternative, filling in the App form by hand and downloading
// the key, is where setups most often go wrong.
func createGitHubApp(ctx context.Context, req githubAppRequest, open func(string) error, out io.Writer) (createdGitHubApp, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return createdGitHubApp{}, err
	}
	defer listener.Close()
	base := "http://" + listener.Addr().String()
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return createdGitHubApp{}, err
	}
	state := hex.EncodeToString(stateBytes)

	manifest, err := json.Marshal(map[string]any{
		"name":         req.name,
		"url":          "https://github.com/n-seiji/henge",
		"redirect_url": base + "/callback",
		"public":       false,
		// Only what the work needs: push branches, open and update pull
		// requests and issues, and read CI results for watched pull requests.
		"default_permissions": map[string]string{
			"contents":      "write",
			"pull_requests": "write",
			"issues":        "write",
			"checks":        "read",
			"statuses":      "read",
			"actions":       "read",
		},
	})
	if err != nil {
		return createdGitHubApp{}, err
	}
	action := req.web + "/settings/apps/new"
	if req.owner != "" {
		action = req.web + "/organizations/" + url.PathEscape(req.owner) + "/settings/apps/new"
	}
	action += "?state=" + state

	results := make(chan createdGitHubApp, 1)
	failures := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		_ = appFormPage.Execute(w, map[string]string{"Action": action, "Manifest": string(manifest)})
	})
	mux.HandleFunc("GET /callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "unexpected state", http.StatusBadRequest)
			return
		}
		app, err := convertManifestCode(r.Context(), req.api, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "henge could not finish creating the App; see the terminal", http.StatusBadGateway)
			failures <- err
			return
		}
		fmt.Fprintln(w, "The GitHub App was created. You can close this tab and return to the terminal.")
		results <- app
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go server.Serve(listener)
	defer server.Close()

	fmt.Fprintf(out, "Opening %s to create the GitHub App. If no browser opens, open it yourself.\n", base)
	if err := open(base); err != nil {
		fmt.Fprintf(out, "  could not open a browser: %v\n", err)
	}
	select {
	case app := <-results:
		return app, nil
	case err := <-failures:
		return createdGitHubApp{}, err
	case <-ctx.Done():
		return createdGitHubApp{}, fmt.Errorf("waiting for GitHub: %w", ctx.Err())
	}
}

func convertManifestCode(ctx context.Context, api, code string) (createdGitHubApp, error) {
	if code == "" {
		return createdGitHubApp{}, errors.New("GitHub returned no code")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, api+"/app-manifests/"+url.PathEscape(code)+"/conversions", nil)
	if err != nil {
		return createdGitHubApp{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return createdGitHubApp{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return createdGitHubApp{}, fmt.Errorf("GitHub answered %s: %s", response.Status, body)
	}
	var app createdGitHubApp
	if err := json.NewDecoder(response.Body).Decode(&app); err != nil {
		return createdGitHubApp{}, err
	}
	if app.ID == 0 || app.PEM == "" {
		return createdGitHubApp{}, errors.New("GitHub returned no App ID or private key")
	}
	return app, nil
}

func openBrowser(target string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	return exec.Command(name, target).Start()
}
