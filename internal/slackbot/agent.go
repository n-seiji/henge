package slackbot

import (
	"context"
	"log"
	"strings"

	"github.com/n-seiji/henge/internal/state"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

const appHomeMessagesTab = "messages"

type suggestedPrompt struct {
	title   string
	message string
}

func isDirectMessage(channel string) bool {
	return strings.HasPrefix(channel, "D")
}

// A direct message involves only its user, so the user's own permission covers
// it.
func (b *Bot) conversationAllowed(channel string) bool {
	return isDirectMessage(channel) || b.channelAllowed(channel)
}

func (b *Bot) handleDirectMessage(ctx context.Context, event *slackevents.MessageEvent) {
	if event.User == "" ||
		event.User == b.config.BotUserID ||
		event.BotID != "" ||
		event.SubType != "" ||
		event.IsEdited() ||
		event.DeletedTimeStamp != "" ||
		!isDirectMessage(event.Channel) {
		return
	}
	message := stripBotMention(event.Text, b.config.BotUserID)
	if message == "" {
		return
	}
	threadTS := event.ThreadTimeStamp
	if threadTS == "" {
		threadTS = event.TimeStamp
	}
	if isAliveQuestion(message) {
		b.answerAliveQuestion(ctx, event.Channel, threadTS)
		return
	}
	if !b.userAllowed(event.User) {
		log.Printf("slackbot: rejecting direct message by %q in %q", event.User, event.Channel)
		b.rejectUnapproved(ctx, &slackevents.AppMentionEvent{
			User:            event.User,
			Channel:         event.Channel,
			TimeStamp:       event.TimeStamp,
			ThreadTimeStamp: event.ThreadTimeStamp,
		}, []approvalSubject{{kind: state.ApprovalUser, id: event.User}})
		return
	}
	if isStopCommand(message) {
		b.stopThread(ctx, event.Channel, threadTS, event.TimeStamp)
		return
	}
	b.processTrigger(ctx, processingTrigger{
		source:      messageTrigger,
		authorID:    event.User,
		channel:     event.Channel,
		timestamp:   event.TimeStamp,
		threadTS:    threadTS,
		message:     message,
		threadReply: event.ThreadTimeStamp != "",
	})
}

func (b *Bot) HandleAppHomeOpened(ctx context.Context, event *slackevents.AppHomeOpenedEvent) {
	if event == nil || event.User == "" {
		return
	}
	if event.Tab == appHomeHomeTab {
		b.publishHome(ctx, event.User)
		return
	}
	if event.Tab != appHomeMessagesTab || event.Channel == "" || !b.userAllowed(event.User) {
		return
	}
	if err := b.api.SetSuggestedPrompts(ctx, event.Channel, b.text().suggestedPrompts); err != nil {
		log.Printf("slackbot: set suggested prompts for %q: %v", event.Channel, err)
	}
}

// Needs the assistant:write scope and the Agent App setting.
func (w *webAPI) SetSuggestedPrompts(ctx context.Context, channel string, prompts []suggestedPrompt) error {
	params := slack.AssistantThreadsSetSuggestedPromptsParameters{ChannelID: channel}
	for _, prompt := range prompts {
		params.AddPrompt(prompt.title, prompt.message)
	}
	return w.client.SetAssistantThreadsSuggestedPromptsContext(ctx, params)
}
