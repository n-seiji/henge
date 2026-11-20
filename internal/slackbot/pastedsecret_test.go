package slackbot

import (
	"context"
	"strings"
	"testing"
)

func TestPastedSecretIsNotSentToCodex(t *testing.T) {
	// Built at run time so the repository holds no token-shaped literal.
	githubToken := "gh" + "p_" + strings.Repeat("A1b2", 9)
	tests := []struct {
		name    string
		text    string
		stopped bool
	}{
		{name: "token in the request", text: "<@UBOT> これで push して " + githubToken, stopped: true},
		{name: "private key in the request", text: "<@UBOT> -----BEGIN OPENSSH " + "PRIVATE KEY-----\nabc", stopped: true},
		{name: "hashes and IDs are ordinary", text: "<@UBOT> 3f9c2a1e7b4d5c6f8a9b0c1d2e3f4a5b6c7d8e9f と 0f8fad5b-d9cb-469f-a165-70867728950e を調べて"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &fakeSlack{}
			runner := completedRunner()
			bot := newTestBot(t, &fakeStore{claim: true}, api, runner)
			bot.config.Workspaces = &fakeWorkspaces{}
			event := mention()
			event.Text = test.text

			bot.HandleMention(context.Background(), event)

			if stopped := len(runner.prompts) == 0; stopped != test.stopped {
				t.Fatalf("turns = %d, want stopped %v", len(runner.prompts), test.stopped)
			}
			if test.stopped && (len(api.postTexts) != 1 || api.postTexts[0] != japanese.pastedSecret) {
				t.Fatalf("posts = %q, want the pasted secret notice", api.postTexts)
			}
		})
	}
}
