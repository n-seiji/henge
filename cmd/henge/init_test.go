package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/n-seiji/henge"
	"github.com/n-seiji/henge/internal/config"
)

type scriptedPrompter struct{ answers []string }

func (p *scriptedPrompter) next(string) (string, error) {
	if len(p.answers) == 0 {
		return "", io.EOF
	}
	answer := p.answers[0]
	p.answers = p.answers[1:]
	return answer, nil
}

func (p *scriptedPrompter) Line(q string) (string, error)   { return p.next(q) }
func (p *scriptedPrompter) Secret(q string) (string, error) { return p.next(q) }

func TestInitAgentDirWritesPrivateTemplatesAndManifest(t *testing.T) {
	dir := t.TempDir()

	if err := initAgentDir(dir, nil, io.Discard); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{".env", "henge.yaml"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("%s permissions = %o, want 600", name, perm)
		}
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "slack-app-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(manifest, henge.SlackAppManifest) {
		t.Error("slack-app-manifest.json differs from the embedded manifest")
	}
}

func TestInitAgentDirKeepsExistingEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	configured := []byte("SLACK_BOT_TOKEN=xoxb-configured\n")
	if err := os.WriteFile(envPath, configured, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := initAgentDir(dir, &scriptedPrompter{}, io.Discard); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, configured) {
		t.Errorf(".env = %q, want it unchanged", got)
	}
}

func TestInitAgentDirAnswersStartAConfigWithTheOwnerChannelAsApprovalChannel(t *testing.T) {
	dir := t.TempDir()
	repo := t.TempDir()
	prompter := &scriptedPrompter{answers: []string{
		"xoxb-1", "xapp-1",
		"me", "U1", // an answer that is not a member ID is asked again
		"C1",
		repo,
		"n",
	}}

	if err := initAgentDir(dir, prompter, io.Discard); err != nil {
		t.Fatal(err)
	}

	env, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "SLACK_BOT_TOKEN=xoxb-1\nSLACK_APP_TOKEN=xapp-1\n"; string(env) != want {
		t.Errorf(".env = %q, want only the tokens %q", env, want)
	}
	t.Chdir(dir)
	t.Setenv("CODEX_HOME", filepath.Join(dir, "codex-home"))
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v, want nil", err)
	}
	if cfg.ApprovalChannelID != "C1" || !slices.Equal(cfg.ApproverUserIDs, []string{"U1"}) || cfg.AdminUserID != "U1" {
		t.Errorf("approval = %q %v admin %q, want C1 [U1] U1", cfg.ApprovalChannelID, cfg.ApproverUserIDs, cfg.AdminUserID)
	}
	if len(cfg.WritableRoots) != 1 {
		t.Errorf("WritableRoots = %v, want the answered repository", cfg.WritableRoots)
	}
}

func TestInitAgentDirStoresTheCreatedGitHubAppKeyPrivately(t *testing.T) {
	dir := t.TempDir()
	previous := newGitHubApp
	t.Cleanup(func() { newGitHubApp = previous })
	newGitHubApp = func(context.Context, githubAppRequest, io.Writer) (createdGitHubApp, error) {
		return createdGitHubApp{ID: 42, PEM: "test-key", HTMLURL: "https://github.com/apps/x"}, nil
	}
	prompter := &scriptedPrompter{answers: []string{"xoxb-1", "xapp-1", "U1", "C1", "", "y", "", ""}}

	if err := initAgentDir(dir, prompter, io.Discard); err != nil {
		t.Fatal(err)
	}

	key, err := os.Stat(filepath.Join(dir, "github-app.pem"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := key.Mode().Perm(); perm != 0o600 {
		t.Errorf("github-app.pem permissions = %o, want 600", perm)
	}
	env, _ := os.ReadFile(filepath.Join(dir, ".env"))
	if !strings.Contains(string(env), "HENGE_GITHUB_APP_PRIVATE_KEY=github-app.pem\n") {
		t.Errorf(".env = %q, want the key file named", env)
	}
	settings, _ := os.ReadFile(filepath.Join(dir, "henge.yaml"))
	if !strings.Contains(string(settings), `app_id: "42"`) {
		t.Errorf("henge.yaml = %q, want app_id 42", settings)
	}
}
