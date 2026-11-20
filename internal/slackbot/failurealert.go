package slackbot

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	// Repeated failures mean something every request needs is broken, not that
	// one request was hard.
	repeatedFailureCount  = 3
	repeatedFailureWindow = time.Hour
	maxFailureAlertRunes  = 300
)

// A broken setting or environment fails every request with the same generic
// reply, so without this nobody who can fix it hears about it.
func (b *Bot) noteWorkFailure(ctx context.Context, err error) {
	if b.config.ApprovalChannelID == "" {
		return
	}
	now := b.now()
	b.failureMu.Lock()
	recent := b.recentFailures[:0]
	for _, at := range b.recentFailures {
		if now.Sub(at) < repeatedFailureWindow {
			recent = append(recent, at)
		}
	}
	b.recentFailures = append(recent, now)
	count := len(b.recentFailures)
	b.failureMu.Unlock()
	if count < repeatedFailureCount {
		return
	}
	b.alertAdmin(ctx, "work", fmt.Sprintf(b.text().repeatedFailureAlert, int(repeatedFailureWindow/time.Minute), count, failureSummary(b.text(), err)))
}

func failureSummary(messages *messages, err error) string {
	if err == nil {
		return messages.unknownCause
	}
	text, _, _ := strings.Cut(err.Error(), "\n")
	text = strings.ReplaceAll(text, "`", "'")
	if runes := []rune(text); len(runes) > maxFailureAlertRunes {
		text = string(runes[:maxFailureAlertRunes]) + "…"
	}
	return text
}
