package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ebix "github.com/n-seiji/ebi-x"
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

func TestInitAgentDirWritesPrivateEnvAndManifest(t *testing.T) {
	dir := t.TempDir()

	if err := initAgentDir(dir, nil, io.Discard); err != nil {
		t.Fatal(err)
	}

	env, err := os.Stat(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := env.Mode().Perm(); perm != 0o600 {
		t.Errorf(".env permissions = %o, want 600", perm)
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "slack-app-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(manifest, ebix.SlackAppManifest) {
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

func TestInitAgentDirMakesTheOwnerChannelTheApprovalChannel(t *testing.T) {
	dir := t.TempDir()
	prompter := &scriptedPrompter{answers: []string{
		"xoxb-1", "xapp-1",
		"seiji", "U1", // an answer that is not a member ID is asked again
		"C1",
	}}

	if err := initAgentDir(dir, prompter, io.Discard); err != nil {
		t.Fatal(err)
	}

	env, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\nSLACK_BOT_TOKEN=xoxb-1\n",
		"\nSLACK_APP_TOKEN=xapp-1\n",
		"\nSLACK_ALLOWED_USER_IDS=U1\n",
		"\nSLACK_APPROVAL_CHANNEL_ID=C1\n",
		"\nSLACK_APPROVER_USER_IDS=U1\n",
		"\nSLACK_ADMIN_USER_ID=U1\n",
	} {
		if !strings.Contains("\n"+string(env), want) {
			t.Errorf(".env lacks %q", strings.TrimSpace(want))
		}
	}
}
