package slackbot

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/n-seiji/henge/internal/codex"
	"github.com/slack-go/slack/slackevents"
)

func bareThreadMention() *slackevents.AppMentionEvent {
	event := mention()
	event.Text = "<@UBOT> "
	event.ThreadTimeStamp = "100.1"
	event.TimeStamp = "100.5"
	return event
}

func TestBareMentionInThreadAnswersTheThread(t *testing.T) {
	api := &fakeSlack{threadMessages: []ThreadMessage{
		{AuthorID: "U2", Timestamp: "100.1", Text: "a question"},
		{AuthorID: "U1", Timestamp: "100.5", Text: "<@UBOT>"},
	}}
	runner := &fakeRunner{responses: []runnerResponse{{result: &codex.TurnResult{Completed: true, Messages: []string{"Done."}}}}}

	newTestBot(t, &fakeStore{claim: true}, api, runner).HandleMention(context.Background(), bareThreadMention())

	if len(runner.prompts) != 1 {
		t.Fatalf("runner prompts = %d, want 1", len(runner.prompts))
	}
	for _, want := range []string{bareThreadMentionRequest, "a question"} {
		if !strings.Contains(runner.prompts[0], want) {
			t.Errorf("turn prompt does not contain %q", want)
		}
	}
}

func TestBareMentionOutsideThreadStartsNothing(t *testing.T) {
	runner := &fakeRunner{}
	event := mention()
	event.Text = "<@UBOT> "

	newTestBot(t, &fakeStore{claim: true}, &fakeSlack{}, runner).HandleMention(context.Background(), event)

	if runner.calls != 0 {
		t.Fatalf("runner calls = %d, want 0", runner.calls)
	}
}

func TestBareThreadMentionStartsOnceApproved(t *testing.T) {
	runner := successfulTurnRunner()
	bot, api, _, _ := newApprovalBot(t, runner)
	bot.now = func() time.Time { return time.Unix(100, 0) }
	event := bareThreadMention()
	event.User = "U2"
	bot.HandleMention(context.Background(), event)

	approvalCommand(bot, api, "UOWNER", "top-1", "許可")

	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}
