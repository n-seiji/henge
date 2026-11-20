package slackbot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/n-seiji/henge/internal/codex"
	"github.com/n-seiji/henge/internal/state"
)

const (
	maxSchedulesPerUser  = 5
	maxScheduleTaskRunes = 500
	scheduleRunPrefix    = "(This is work the requester scheduled to run %s. Earlier runs' results are not in this thread.)\n"
)

// Only a person's request may change a schedule, so unattended turns cannot
// start more unattended work.
func (b *Bot) applySchedule(trigger processingTrigger, output workOutput) string {
	if !output.scheduleInvalid && output.schedule == nil {
		return ""
	}
	if trigger.source != mentionTrigger && trigger.source != messageTrigger {
		log.Printf("slackbot: ignore schedule from an unattended turn in %s:%s", trigger.channel, trigger.threadTS)
		return b.text().scheduleSourceNotice
	}
	if output.scheduleInvalid {
		return b.text().scheduleInvalidNotice
	}
	threadKey := threadRef(trigger.channel, trigger.threadTS)
	request := output.schedule
	if request.Stop {
		previous, deleted, err := b.store.DeleteSchedule(threadKey)
		if err != nil {
			log.Printf("slackbot: stop schedule %q: %v", threadKey, err)
			return b.text().scheduleInvalidNotice
		}
		if !deleted {
			return ""
		}
		return fmt.Sprintf(b.text().scheduleStopNotice, previous.Spec)
	}
	spec, err := codex.ParseScheduleSpec(request.When)
	if err != nil {
		log.Printf("slackbot: reject schedule %q in %q: %v", request.When, threadKey, err)
		return fmt.Sprintf(b.text().scheduleSpecNotice, shortDetail(request.When))
	}
	others := 0
	for _, schedule := range b.store.Schedules() {
		if schedule.AuthorID == trigger.authorID && threadRef(schedule.Channel, schedule.ThreadTS) != threadKey {
			others++
		}
	}
	if others >= maxSchedulesPerUser {
		return fmt.Sprintf(b.text().scheduleLimitNotice, maxSchedulesPerUser)
	}
	now := b.now()
	schedule := state.Schedule{
		Channel:   trigger.channel,
		ThreadTS:  trigger.threadTS,
		AuthorID:  trigger.authorID,
		Spec:      spec.String(),
		Task:      clipText(request.Task, maxScheduleTaskRunes),
		NextAt:    spec.Next(now.In(time.Local)),
		CreatedAt: now,
	}
	if err := b.store.SetSchedule(threadKey, schedule); err != nil {
		log.Printf("slackbot: save schedule %q: %v", threadKey, err)
		return b.text().scheduleInvalidNotice
	}
	return fmt.Sprintf(b.text().scheduleSetNotice, schedule.Spec, schedule.Task, formatFollowUpTime(schedule.NextAt))
}

func (b *Bot) nextOccurrence(schedule state.Schedule) time.Time {
	spec, err := codex.ParseScheduleSpec(schedule.Spec)
	if err != nil {
		// Specs are saved in canonical form, so this means a hand-edited
		// file. Park it rather than run it on every poll.
		log.Printf("slackbot: unreadable schedule %q in %s:%s: %v", schedule.Spec, schedule.Channel, schedule.ThreadTS, err)
		return b.now().AddDate(100, 0, 0)
	}
	return spec.Next(b.now().In(time.Local))
}

func (b *Bot) runSchedules(acceptCtx, turnCtx context.Context, wg *sync.WaitGroup) {
	for {
		due, err := b.store.TakeDueSchedules(b.now(), b.nextOccurrence)
		if err != nil {
			log.Printf("slackbot: take due schedules: %v", err)
		}
		for _, schedule := range due {
			wg.Go(func() { b.fireSchedule(turnCtx, schedule) })
		}
		if err := b.sleep(acceptCtx, followUpPollInterval); err != nil {
			return
		}
	}
}

// Access is checked again, since the user or channel may have lost it since.
func (b *Bot) fireSchedule(ctx context.Context, schedule state.Schedule) {
	threadKey := threadRef(schedule.Channel, schedule.ThreadTS)
	if !b.userAllowed(schedule.AuthorID) || !b.conversationAllowed(schedule.Channel) {
		log.Printf("slackbot: skip schedule %q: user %q or channel is no longer allowed", threadKey, schedule.AuthorID)
		return
	}
	text := fmt.Sprintf(b.text().scheduleStartMessage, schedule.Spec, schedule.Task)
	if link, err := b.api.Permalink(ctx, schedule.Channel, schedule.ThreadTS); err == nil {
		text += fmt.Sprintf(b.text().scheduleOriginLink, link)
	} else {
		log.Printf("slackbot: link schedule thread %q: %v", threadKey, err)
	}
	var timestamp string
	if err := b.retrySlack(ctx, func() error {
		var err error
		timestamp, err = b.api.PostMessage(ctx, schedule.Channel, "", text)
		return err
	}); err != nil {
		log.Printf("slackbot: announce schedule %q: %v", threadKey, err)
		return
	}
	b.processTrigger(ctx, processingTrigger{
		source:    scheduleTrigger,
		authorID:  schedule.AuthorID,
		channel:   schedule.Channel,
		timestamp: timestamp,
		threadTS:  timestamp,
		message:   fmt.Sprintf(scheduleRunPrefix, schedule.Spec) + schedule.Task,
	})
}

func (b *Bot) stopSchedule(channel, threadTS string) bool {
	_, deleted, err := b.store.DeleteSchedule(threadRef(channel, threadTS))
	if err != nil {
		log.Printf("slackbot: stop schedule %s:%s: %v", channel, threadTS, err)
	}
	return deleted
}

func scheduleSummary(text *messages, schedule state.Schedule) string {
	return fmt.Sprintf(text.scheduleSummary, schedule.Spec, clipText(schedule.Task, 100), formatFollowUpTime(schedule.NextAt))
}
