package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"
)

var formAction = regexp.MustCompile(`action="([^"]+)"`)

func TestCreateGitHubAppReturnsWhatGitHubConvertsTheCodeInto(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/app-manifests/the-code/conversions" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":7,"slug":"henge-x","pem":"KEY","html_url":"https://github.com/apps/henge-x"}`)
	}))
	defer api.Close()

	// Plays the browser and GitHub: reads the form henge serves, then comes
	// back to the callback with the state GitHub would echo.
	browser := func(page string) error {
		go func() {
			response, err := http.Get(page)
			if err != nil {
				return
			}
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			match := formAction.FindSubmatch(body)
			if match == nil {
				return
			}
			action, _ := url.Parse(string(match[1]))
			state := action.Query().Get("state")
			callback, _ := http.Get(page + "/callback?code=the-code&state=" + url.QueryEscape(state))
			if callback != nil {
				callback.Body.Close()
			}
		}()
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app, err := createGitHubApp(ctx, githubAppRequest{name: "henge-x", web: "https://github.example", api: api.URL}, browser, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if app.ID != 7 || app.PEM != "KEY" || app.HTMLURL != "https://github.com/apps/henge-x" {
		t.Errorf("app = %+v, want ID 7 with the key and URL GitHub returned", app)
	}
}
