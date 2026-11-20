package slackbot

import "strings"

// Mentions them so they are notified even when they left the thread.
func (b *Bot) waitingNotice(authorID, channel, threadTS string, questions []string) string {
	var builder strings.Builder
	builder.WriteString("🙋 ")
	if authorID != "" && b.userAllowed(authorID) {
		builder.WriteString("<@" + authorID + "> ")
	}
	text := b.text()
	builder.WriteString(text.waitingPrefix)
	if b.repliesWithoutMention(channel, threadTS) {
		builder.WriteString(text.waitingReply)
	} else {
		builder.WriteString(text.waitingMention)
	}
	for _, question := range questions {
		builder.WriteString("\n- " + question)
	}
	return builder.String()
}

func (b *Bot) repliesWithoutMention(channel, threadTS string) bool {
	if isDirectMessage(channel) {
		return true
	}
	subscription, ok := b.store.GetSubscription(threadRef(channel, threadTS))
	return ok && subscription.ExpiresAt.After(b.now())
}
