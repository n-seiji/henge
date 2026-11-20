package slackbot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/n-seiji/henge/internal/codex"
	"github.com/n-seiji/henge/internal/state"
)

const (
	followUpPollInterval = 30 * time.Second
	maxFollowUpTaskRunes = 500
	followUpTimeLayout   = "1/2 15:04 MST"
)

func (b *Bot) pendingFollowUp(threadKey string) string {
	followUp, ok := b.store.GetFollowUp(threadKey)
	if !ok {
		return ""
	}
	return formatFollowUpTime(followUp.DueAt) + ": " + followUp.Task
}

// A turn that sets no follow-up cancels the pending one, so a follow-up never
// outlives the plan that made it.
func (b *Bot) scheduleFollowUp(trigger processingTrigger, output workOutput) string {
	threadKey := threadRef(trigger.channel, trigger.threadTS)
	next, notice := b.nextFollowUp(trigger, output)
	if next != nil {
		if err := b.store.SetFollowUp(threadKey, *next); err != nil {
			log.Printf("slackbot: save follow-up %q: %v", threadKey, err)
			return b.text().followUpInvalidNotice
		}
		return fmt.Sprintf(b.text().followUpScheduledNotice, formatFollowUpTime(next.DueAt), next.Task)
	}
	// A due follow-up was taken from the store before it ran, so only a
	// person's request can still have one pending. A pull request event
	// leaves it alone: it did not come from the plan that made it.
	if trigger.source == mentionTrigger || trigger.source == messageTrigger {
		previous, deleted, err := b.store.DeleteFollowUp(threadKey)
		if err != nil {
			log.Printf("slackbot: cancel follow-up %q: %v", threadKey, err)
		}
		if deleted {
			notice = strings.TrimSpace(fmt.Sprintf(b.text().followUpCancelledNotice, formatFollowUpTime(previous.DueAt)) + "\n" + notice)
		}
	}
	return notice
}

func (b *Bot) nextFollowUp(trigger processingTrigger, output workOutput) (*state.FollowUp, string) {
	if output.followUpInvalid {
		return nil, b.text().followUpInvalidNotice
	}
	if output.followUp == nil {
		return nil, ""
	}
	chain := 1
	if trigger.followUp != nil {
		chain = trigger.followUp.Chain + 1
	}
	if chain > codex.MaxFollowUpChain {
		return nil, fmt.Sprintf(b.text().followUpChainNotice, codex.MaxFollowUpChain)
	}
	now := b.now()
	dueAt, err := codex.FollowUpDueAt(output.followUp.When, now)
	if err != nil {
		log.Printf("slackbot: reject follow-up time %q in %s:%s: %v", output.followUp.When, trigger.channel, trigger.threadTS, err)
		return nil, fmt.Sprintf(b.text().followUpTimeNotice, shortDetail(output.followUp.When), codex.FollowUpRange())
	}
	return &state.FollowUp{
		Channel:   trigger.channel,
		ThreadTS:  trigger.threadTS,
		AuthorID:  trigger.authorID,
		Task:      clipText(output.followUp.Task, maxFollowUpTaskRunes),
		DueAt:     dueAt,
		Chain:     chain,
		CreatedAt: now,
	}, ""
}

func formatFollowUpTime(at time.Time) string {
	return at.In(time.Local).Format(followUpTimeLayout)
}

func (b *Bot) runFollowUps(acceptCtx, turnCtx context.Context, wg *sync.WaitGroup) {
	for {
		due, err := b.store.TakeDueFollowUps(b.now())
		if err != nil {
			log.Printf("slackbot: take due follow-ups: %v", err)
		}
		for _, followUp := range due {
			wg.Go(func() { b.fireFollowUp(turnCtx, followUp) })
		}
		if err := b.sleep(acceptCtx, followUpPollInterval); err != nil {
			return
		}
	}
}

// Access is checked again, since the user or channel may have lost it in the
// meantime.
func (b *Bot) fireFollowUp(ctx context.Context, followUp state.FollowUp) {
	threadKey := threadRef(followUp.Channel, followUp.ThreadTS)
	if !b.userAllowed(followUp.AuthorID) || !b.conversationAllowed(followUp.Channel) {
		log.Printf("slackbot: drop follow-up %q: user %q or channel is no longer allowed", threadKey, followUp.AuthorID)
		return
	}
	var timestamp string
	if err := b.retrySlack(ctx, func() error {
		var err error
		timestamp, err = b.api.PostMessage(ctx, followUp.Channel, followUp.ThreadTS, fmt.Sprintf(b.text().followUpStartMessage, followUp.Task))
		return err
	}); err != nil {
		log.Printf("slackbot: announce follow-up %q: %v", threadKey, err)
		return
	}
	b.processTrigger(ctx, processingTrigger{
		source:      followUpTrigger,
		authorID:    followUp.AuthorID,
		channel:     followUp.Channel,
		timestamp:   timestamp,
		threadTS:    followUp.ThreadTS,
		message:     followUp.Task,
		threadReply: true,
		followUp:    &followUp,
	})
}
