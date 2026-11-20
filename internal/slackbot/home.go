package slackbot

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

const (
	appHomeHomeTab = "home"
	// maxHomeItems bounds each list on the Home tab, which Slack limits in
	// size.
	maxHomeItems = 15
)

type homeThread struct {
	channel  string
	threadTS string
	detail   string
	at       time.Time
}

func (b *Bot) publishHome(ctx context.Context, userID string) {
	if !b.userAllowed(userID) {
		if err := b.api.PublishHome(ctx, userID, []string{b.text().homeForbidden}); err != nil {
			log.Printf("slackbot: publish home for %q: %v", userID, err)
		}
		return
	}
	var running []homeThread
	b.runningMu.Lock()
	for threadKey, requests := range b.running {
		channel, threadTS, ok := strings.Cut(threadKey, ":")
		if !ok {
			continue
		}
		for _, request := range requests {
			if request.authorID == userID {
				running = append(running, homeThread{channel: channel, threadTS: threadTS, at: request.startedAt})
				break
			}
		}
	}
	b.runningMu.Unlock()
	sort.Slice(running, func(i, j int) bool { return running[i].at.Before(running[j].at) })

	var followUps []homeThread
	for _, followUp := range b.store.FollowUps() {
		if followUp.AuthorID == userID {
			followUps = append(followUps, homeThread{channel: followUp.Channel, threadTS: followUp.ThreadTS, detail: followUp.Task, at: followUp.DueAt})
		}
	}
	var watches []homeThread
	for _, watch := range b.store.PullWatches() {
		if watch.AuthorID == userID {
			detail := watch.URL
			if label := b.text().checksLabels[watch.Checks]; label != "" {
				detail += fmt.Sprintf(b.text().homeCI, label)
			}
			watches = append(watches, homeThread{channel: watch.Channel, threadTS: watch.ThreadTS, detail: detail})
		}
	}

	var schedules []homeThread
	for _, schedule := range b.store.Schedules() {
		if schedule.AuthorID == userID {
			schedules = append(schedules, homeThread{channel: schedule.Channel, threadTS: schedule.ThreadTS, detail: scheduleSummary(b.text(), schedule)})
		}
	}

	var reminders []homeThread
	for _, reminder := range b.store.Reminders() {
		if reminder.AuthorID == userID {
			reminders = append(reminders, homeThread{channel: reminder.Channel, threadTS: reminder.ThreadTS, detail: reminder.Text, at: reminder.DueAt})
		}
	}

	var doneWatches []homeThread
	for _, watch := range b.store.DoneWatches() {
		if watch.AuthorID == userID {
			doneWatches = append(doneWatches, homeThread{channel: watch.Channel, threadTS: watch.ThreadTS, detail: doneWatchSummary(b.text(), watch)})
		}
	}

	text := b.text()
	links := make(map[string]string)
	sections := []string{text.homeTitle}
	sections = append(sections, b.homeSection(ctx, text, links, text.homeRunning, running, func(item homeThread) string {
		return fmt.Sprintf(text.homeRunningSince, formatFollowUpTime(item.at))
	}))
	sections = append(sections, b.homeSection(ctx, text, links, text.homeFollowUps, followUps, func(item homeThread) string {
		return fmt.Sprintf(text.homeFollowUpAt, formatFollowUpTime(item.at), clipText(item.detail, 100))
	}))
	sections = append(sections, b.homeSection(ctx, text, links, text.homeWatches, watches, func(item homeThread) string {
		return item.detail
	}))
	sections = append(sections, b.homeSection(ctx, text, links, text.homeSchedules, schedules, func(item homeThread) string {
		return item.detail
	}))
	sections = append(sections, b.homeSection(ctx, text, links, text.homeReminders, reminders, func(item homeThread) string {
		return formatFollowUpTime(item.at) + ": " + clipText(item.detail, 100)
	}))
	sections = append(sections, b.homeSection(ctx, text, links, text.homeDoneWatches, doneWatches, func(item homeThread) string {
		return item.detail
	}))
	sections = append(sections, homeUserSettings(text, b.readUserSettings(userID)))
	sections = append(sections, fmt.Sprintf(text.homeUpdated, formatFollowUpTime(b.now())))
	if err := b.api.PublishHome(ctx, userID, sections); err != nil {
		log.Printf("slackbot: publish home for %q: %v", userID, err)
	}
}

// The Home tab is the one place settings are shown, since only its owner sees
// it.
func homeUserSettings(text *messages, settings []string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, text.homeCount, text.homeSettingsTitle, len(settings))
	if len(settings) == 0 {
		builder.WriteString("\n" + text.homeEmpty)
	}
	for _, setting := range settings {
		builder.WriteString("\n• " + setting)
	}
	return builder.String() + "\n_" + text.homeSettingsHelp + "_"
}

func (b *Bot) homeSection(ctx context.Context, text *messages, links map[string]string, title string, items []homeThread, describe func(homeThread) string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, text.homeCount, title, len(items))
	if len(items) == 0 {
		return builder.String() + "\n" + text.homeEmpty
	}
	for i, item := range items {
		if i >= maxHomeItems {
			fmt.Fprintf(&builder, text.homeMore, len(items)-i)
			break
		}
		fmt.Fprintf(&builder, "\n• %s — %s", b.threadLink(ctx, text, links, item.channel, item.threadTS), describe(item))
	}
	return builder.String()
}

func (b *Bot) threadLink(ctx context.Context, text *messages, links map[string]string, channel, threadTS string) string {
	key := threadRef(channel, threadTS)
	if link, ok := links[key]; ok {
		return link
	}
	place := fmt.Sprintf(text.homeChannelThread, channel)
	if isDirectMessage(channel) {
		place = text.homeDMThread
	}
	link := place
	if permalink, err := b.api.Permalink(ctx, channel, threadTS); err == nil && permalink != "" {
		link = "<" + permalink + "|" + place + ">"
	} else if err != nil {
		log.Printf("slackbot: permalink %s: %v", key, err)
	}
	links[key] = link
	return link
}

func (w *webAPI) PublishHome(ctx context.Context, userID string, sections []string) error {
	var blocks []slack.Block
	for i, section := range sections {
		if i > 0 {
			blocks = append(blocks, slack.NewDividerBlock())
		}
		blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, section, false, false), nil, nil))
	}
	_, err := w.client.PublishViewContext(ctx, slack.PublishViewContextRequest{
		UserID: userID,
		View:   slack.HomeTabViewRequest{Type: slack.VTHomeTab, Blocks: slack.Blocks{BlockSet: blocks}},
	})
	return err
}

func (w *webAPI) Permalink(ctx context.Context, channel, timestamp string) (string, error) {
	return w.client.GetPermalinkContext(ctx, &slack.PermalinkParameters{Channel: channel, Ts: timestamp})
}
