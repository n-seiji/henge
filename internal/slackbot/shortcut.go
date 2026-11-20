package slackbot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/slack-go/slack"
)

const (
	requestShortcutID = "henge_request"
	requestModalID    = "henge_request_modal"
	// Slack apps created from the manifest before the rename keep sending the
	// old shortcut ID until an admin edits it, and a form opened before the
	// upgrade submits the old modal ID; dropping them would silently break
	// the shortcut on those workspaces.
	legacyRequestShortcutID = "ebix_request"
	legacyRequestModalID    = "ebix_request_modal"
	requestBlockID          = "request"
	requestActionID         = "request_text"
	maxQuotedRunes          = 500
)

type requestTarget struct {
	Channel   string `json:"channel"`
	MessageTS string `json:"message_ts"`
	ThreadTS  string `json:"thread_ts"`
}

func (t requestTarget) thread() string {
	if t.ThreadTS != "" {
		return t.ThreadTS
	}
	return t.MessageTS
}

func (b *Bot) HandleInteraction(ctx context.Context, callback *slack.InteractionCallback) {
	switch {
	case callback.Type == slack.InteractionTypeMessageAction && (callback.CallbackID == requestShortcutID || callback.CallbackID == legacyRequestShortcutID):
		b.HandleRequestShortcut(ctx, callback)
	case callback.Type == slack.InteractionTypeViewSubmission && (callback.View.CallbackID == requestModalID || callback.View.CallbackID == legacyRequestModalID):
		b.HandleRequestSubmission(ctx, callback)
	}
}

// Only users who may use the bot in that conversation get it.
func (b *Bot) HandleRequestShortcut(ctx context.Context, callback *slack.InteractionCallback) {
	target := requestTarget{Channel: callback.Channel.ID, MessageTS: callback.Message.Timestamp, ThreadTS: callback.Message.ThreadTimestamp}
	if target.Channel == "" || target.MessageTS == "" {
		return
	}
	if !b.userAllowed(callback.User.ID) || !b.channelAllowed(target.Channel) {
		if err := b.api.PostEphemeral(ctx, target.Channel, callback.User.ID, b.text().shortcutForbidden); err != nil {
			log.Printf("slackbot: post shortcut refusal: %v", err)
		}
		return
	}
	metadata, err := json.Marshal(target)
	if err != nil {
		log.Printf("slackbot: encode request target: %v", err)
		return
	}
	quoted := clipText(callback.Message.Text, maxQuotedRunes)
	if err := b.api.OpenRequestModal(ctx, b.text(), callback.TriggerID, string(metadata), quoted); err != nil {
		log.Printf("slackbot: open request modal: %v", err)
	}
}

// Permissions are checked again, since the modal may have stayed open while
// they changed.
func (b *Bot) HandleRequestSubmission(ctx context.Context, callback *slack.InteractionCallback) {
	var target requestTarget
	if err := json.Unmarshal([]byte(callback.View.PrivateMetadata), &target); err != nil || target.Channel == "" || target.MessageTS == "" {
		log.Printf("slackbot: read request target: %v", err)
		return
	}
	user := callback.User.ID
	if !b.userAllowed(user) || !b.channelAllowed(target.Channel) {
		return
	}
	text := strings.TrimSpace(callback.View.State.Values[requestBlockID][requestActionID].Value)
	if text == "" {
		return
	}
	threadTS := target.thread()
	var timestamp string
	if err := b.retrySlack(ctx, func() error {
		var err error
		timestamp, err = b.api.PostMessage(ctx, target.Channel, threadTS, formatRequestPost(b.text(), user, text))
		return err
	}); err != nil {
		log.Printf("slackbot: post shortcut request: %v", err)
		return
	}
	// The thread, including the message the request is about, reaches the
	// session as context, so the request text is sent as written.
	b.processTrigger(ctx, processingTrigger{
		source:      mentionTrigger,
		authorID:    user,
		channel:     target.Channel,
		timestamp:   timestamp,
		threadTS:    threadTS,
		message:     text,
		threadReply: true,
	})
}

func formatRequestPost(messages *messages, user, text string) string {
	return fmt.Sprintf(messages.requestPost, user, strings.ReplaceAll(text, "\n", "\n> "))
}

func (w *webAPI) OpenRequestModal(ctx context.Context, text *messages, triggerID, metadata, quoted string) error {
	input := slack.NewPlainTextInputBlockElement(slack.NewTextBlockObject(slack.PlainTextType, text.requestPlaceholder, false, false), requestActionID)
	input.Multiline = true
	blocks := []slack.Block{
		slack.NewContextBlock("", slack.NewTextBlockObject(slack.PlainTextType, fmt.Sprintf(text.requestTarget, quoted), false, false)),
		slack.NewInputBlock(requestBlockID, slack.NewTextBlockObject(slack.PlainTextType, text.requestLabel, false, false), nil, input),
	}
	_, err := w.client.OpenViewContext(ctx, triggerID, slack.ModalViewRequest{
		Type:            slack.VTModal,
		CallbackID:      requestModalID,
		Title:           slack.NewTextBlockObject(slack.PlainTextType, text.requestTitle, false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, text.requestSubmit, false, false),
		Close:           slack.NewTextBlockObject(slack.PlainTextType, text.requestClose, false, false),
		Blocks:          slack.Blocks{BlockSet: blocks},
		PrivateMetadata: metadata,
	})
	return err
}

func (w *webAPI) PostEphemeral(ctx context.Context, channel, user, text string) error {
	_, err := w.client.PostEphemeralContext(ctx, channel, user, slack.MsgOptionText(text, false))
	return err
}
