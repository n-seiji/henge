package slackbot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/n-seiji/henge/internal/state"
	"github.com/slack-go/slack"
)

const (
	doneWatchPollInterval = 30 * time.Minute
	defaultDoneWatchDays  = 1
	maxDoneWatchDays      = 7
	maxDoneWatchReminders = 5
	maxDoneWatchesPerUser = 10
	maxDoneWatchTargets   = 100
)

var doneWatchCommand = regexp.MustCompile(`^(?:(毎日)|(\d{1,2})日(おき|ごと|毎)に?)?\s*監視(?:して|お願い(?:します)?|よろしく)?(?:ください)?[。．.!！\s]*$`)

var englishDoneWatchCommand = regexp.MustCompile(`(?i)^watch(?: this)?(?: (daily)| every (\d{1,2}) weekdays?)?[.!\s]*$`)

var englishDoneWatchStopCommand = regexp.MustCompile(`(?i)^stop watching[.!\s]*$`)

var doneWatchStopCommand = regexp.MustCompile(`^監視を?(?:とめ|止め|やめ|停止し?|終了し?|解除し?)(?:て(?:ください)?)?[。．.!！\s]*$|^監視(?:停止|終了|解除)[。．.!！\s]*$`)

var slackUserMention = regexp.MustCompile(`<@([UW][A-Z0-9]+)(?:\|[^>]*)?>`)

var slackBroadcast = regexp.MustCompile(`<!(?:channel|here|everyone)(?:\|[^>]*)?>`)

var slackUserGroup = regexp.MustCompile(`<!subteam\^([A-Z0-9]+)(?:\|[^>]*)?>`)

func parseDoneWatchCommand(text, botUserID string) (int, bool) {
	if botUserID == "" {
		return 0, false
	}
	mention := "<@" + botUserID + ">"
	index := strings.LastIndex(text, mention)
	if index < 0 {
		return 0, false
	}
	command := strings.TrimSpace(text[index+len(mention):])
	match := doneWatchCommand.FindStringSubmatch(command)
	if match == nil {
		english := englishDoneWatchCommand.FindStringSubmatch(command)
		if english == nil {
			return 0, false
		}
		match = []string{english[0], english[1], english[2], "ごと"}
	}
	switch {
	case match[1] != "":
		return 1, true
	case match[2] != "":
		n, err := strconv.Atoi(match[2])
		if err != nil || n < 1 {
			return 0, false
		}
		// "N日おき" skips N days; "N日ごと" repeats every N days.
		if match[3] == "おき" {
			n++
		}
		if n > maxDoneWatchDays {
			return 0, false
		}
		return n, true
	}
	return defaultDoneWatchDays, true
}

func isDoneWatchStopCommand(message string) bool {
	return doneWatchStopCommand.MatchString(strings.TrimSpace(message)) || englishDoneWatchStopCommand.MatchString(strings.TrimSpace(message))
}

func mentionedUsers(text, botUserID string) []string {
	seen := make(map[string]struct{})
	var users []string
	for _, match := range slackUserMention.FindAllStringSubmatch(text, -1) {
		id := match[1]
		if id == botUserID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		users = append(users, id)
	}
	return users
}

func (b *Bot) startDoneWatch(ctx context.Context, channel, threadTS, timestamp, text, authorID string, days int) {
	messageTS := timestamp
	targets := mentionedUsers(text, b.config.BotUserID)
	everyone := slackBroadcast.MatchString(text)
	groups := mentionedGroups(text)
	if len(targets) == 0 && !everyone && len(groups) == 0 && threadTS != timestamp {
		messages, err := b.api.GetThreadMessages(ctx, channel, threadTS, threadTS)
		if err != nil {
			log.Printf("slackbot: read thread %s for done watch: %v", threadRef(channel, threadTS), err)
			b.postDoneWatchNotice(ctx, channel, threadTS, b.text().doneWatchFailedNotice)
			return
		}
		for _, message := range messages {
			if message.Timestamp == threadTS {
				messageTS = threadTS
				targets = mentionedUsers(message.Text, b.config.BotUserID)
				everyone = slackBroadcast.MatchString(message.Text)
				groups = mentionedGroups(message.Text)
				break
			}
		}
	}
	outside := 0
	var namedOutside []string
	if everyone || len(groups) > 0 {
		channelMembers, err := b.api.ChannelMembers(ctx, channel)
		if err == nil {
			namedOutside = notIn(targets, channelMembers)
			targets, outside, err = b.expandDoneWatchTargets(ctx, channelMembers, targets, everyone, groups, authorID)
		}
		if err != nil {
			log.Printf("slackbot: list members for done watch in %s: %v", channel, err)
			notice := b.text().doneWatchFailedNotice
			if response, ok := errors.AsType[slack.SlackErrorResponse](err); ok && response.Err == "missing_scope" {
				notice = b.text().doneWatchScopeNotice
			}
			b.postDoneWatchNotice(ctx, channel, threadTS, notice)
			return
		}
	} else if len(targets) > 0 {
		// Named people stay targets even when they are not in the channel,
		// since the author meant them; the start notice says so, so the
		// author can invite them. Without channels:read the note is skipped.
		if channelMembers, err := b.api.ChannelMembers(ctx, channel); err == nil {
			namedOutside = notIn(targets, channelMembers)
		} else {
			log.Printf("slackbot: list members of %s for done watch: %v", channel, err)
		}
	}
	if len(targets) == 0 {
		notice := b.text().doneWatchNoTargetNotice
		if outside > 0 {
			notice = b.text().doneWatchAllOutsideNotice
		}
		b.postDoneWatchNotice(ctx, channel, threadTS, notice)
		return
	}
	if len(targets) > maxDoneWatchTargets {
		b.postDoneWatchNotice(ctx, channel, threadTS, fmt.Sprintf(b.text().doneWatchTooManyNotice, maxDoneWatchTargets))
		return
	}
	key := channel + ":" + messageTS
	others := 0
	for _, watch := range b.store.DoneWatches() {
		if watch.AuthorID == authorID && watch.Key() != key {
			others++
		}
	}
	if others >= maxDoneWatchesPerUser {
		b.postDoneWatchNotice(ctx, channel, threadTS, fmt.Sprintf(b.text().doneWatchLimitNotice, maxDoneWatchesPerUser))
		return
	}
	settings := b.settings()
	reaction, hour := settings.doneReaction(), settings.doneWatchHour()
	reacted, err := b.api.ReactionUsers(ctx, channel, messageTS, reaction)
	if err != nil {
		log.Printf("slackbot: read reactions of %s: %v", key, err)
		b.postDoneWatchNotice(ctx, channel, threadTS, b.text().doneWatchFailedNotice)
		return
	}
	if len(pendingUsers(targets, reacted)) == 0 {
		b.postDoneWatchNotice(ctx, channel, threadTS, withReaction(b.text().doneWatchAllDoneNotice, reaction))
		return
	}
	now := b.now()
	watch := state.DoneWatch{
		Channel: channel, ThreadTS: threadTS, MessageTS: messageTS, AuthorID: authorID,
		Targets: targets, IntervalDays: days, Reaction: reaction,
		NextAt: nextDoneWatchReminder(now, days, hour), CreatedAt: now,
	}
	if err := b.store.SetDoneWatch(watch); err != nil {
		log.Printf("slackbot: save done watch %s: %v", key, err)
		b.postDoneWatchNotice(ctx, channel, threadTS, b.text().doneWatchFailedNotice)
		return
	}
	b.addReaction(ctx, channel, timestamp, "eyes")
	outsideNote := ""
	if outside > 0 {
		outsideNote = fmt.Sprintf(b.text().doneWatchOutsideNotice, outside)
	}
	if len(namedOutside) > 0 {
		outsideNote += fmt.Sprintf(b.text().doneWatchNamedOutsideNote, userMentions(namedOutside))
	}
	b.postDoneWatchNotice(ctx, channel, threadTS, withReaction(fmt.Sprintf(b.text().doneWatchStartedNotice, len(targets), doneWatchIntervalLabel(b.text(), days, hour), formatFollowUpTime(watch.NextAt), outsideNote, maxDoneWatchReminders), reaction))
}

func nextDoneWatchReminder(after time.Time, days, hour int) time.Time {
	local := after.In(time.Local)
	at := time.Date(local.Year(), local.Month(), local.Day(), hour, 0, 0, 0, time.Local)
	if !at.After(local) {
		at = at.AddDate(0, 0, 1)
	}
	left := max(days, 1)
	for {
		if !isWeekend(at) {
			if left--; left == 0 {
				return at
			}
		}
		at = at.AddDate(0, 0, 1)
	}
}

// Only channel members count, since a mention does not reach anyone else.
func (b *Bot) expandDoneWatchTargets(ctx context.Context, channelMembers, named []string, everyone bool, groups []string, authorID string) (targets []string, outside int, err error) {
	inChannel := makeSet(channelMembers)
	var members []string
	if everyone {
		members = append(members, channelMembers...)
	}
	var away []string
	for _, group := range groups {
		groupMembers, err := b.api.UserGroupMembers(ctx, group)
		if err != nil {
			return nil, 0, err
		}
		for _, member := range groupMembers {
			if _, ok := inChannel[member]; ok {
				members = append(members, member)
			} else {
				away = append(away, member)
			}
		}
	}
	// One People call over everyone keeps it to a single users.list walk.
	people, err := b.api.People(ctx, append(members, away...))
	if err != nil {
		return nil, 0, err
	}
	isPerson := makeSet(people)
	var humans []string
	for _, member := range members {
		if _, ok := isPerson[member]; ok {
			humans = append(humans, member)
		}
	}
	targets = mergeTargets(named, humans, authorID, b.config.BotUserID)
	chosen := makeSet(targets)
	counted := map[string]struct{}{}
	for _, member := range away {
		_, person := isPerson[member]
		_, target := chosen[member]
		_, seen := counted[member]
		if person && !target && !seen && member != authorID {
			counted[member] = struct{}{}
		}
	}
	return targets, len(counted), nil
}

func (b *Bot) postDoneWatchNotice(ctx context.Context, channel, threadTS, text string) {
	if err := b.post(ctx, channel, threadTS, text); err != nil {
		log.Printf("slackbot: post done watch notice: %v", err)
	}
}

func doneWatchIntervalLabel(text *messages, days, hour int) string {
	switch days {
	case 1:
		return fmt.Sprintf(text.doneWatchEveryDay, hour)
	case 2:
		return fmt.Sprintf(text.doneWatchEveryOtherDay, hour)
	}
	return fmt.Sprintf(text.doneWatchEveryNDays, days, hour)
}

// Watches saved before the reaction became a setting have none.
func watchReaction(watch state.DoneWatch) string {
	if watch.Reaction == "" {
		return defaultDoneReaction
	}
	return watch.Reaction
}

func withReaction(text, reaction string) string {
	return strings.ReplaceAll(text, ":done:", ":"+reaction+":")
}

func notIn(users, members []string) []string {
	in := makeSet(members)
	var out []string
	for _, user := range users {
		if _, ok := in[user]; !ok {
			out = append(out, user)
		}
	}
	return out
}

func mentionedGroups(text string) []string {
	var groups []string
	seen := map[string]struct{}{}
	for _, match := range slackUserGroup.FindAllStringSubmatch(text, -1) {
		if _, ok := seen[match[1]]; ok {
			continue
		}
		seen[match[1]] = struct{}{}
		groups = append(groups, match[1])
	}
	return groups
}

func mergeTargets(named, members []string, authorID, botUserID string) []string {
	targets := append([]string(nil), named...)
	seen := makeSet(named)
	for _, member := range members {
		if member == authorID || member == botUserID {
			continue
		}
		if _, ok := seen[member]; ok {
			continue
		}
		seen[member] = struct{}{}
		targets = append(targets, member)
	}
	return targets
}

func pendingUsers(targets, reacted []string) []string {
	done := makeSet(reacted)
	var pending []string
	for _, target := range targets {
		if _, ok := done[target]; !ok {
			pending = append(pending, target)
		}
	}
	return pending
}

func userMentions(users []string) string {
	mentions := make([]string, len(users))
	for i, user := range users {
		mentions[i] = "<@" + user + ">"
	}
	return strings.Join(mentions, " ")
}

// Wakes for a reminder due before the next poll, so reminders go out on the
// hour.
func (b *Bot) runDoneWatches(ctx context.Context) {
	for {
		for _, watch := range b.store.DoneWatches() {
			if ctx.Err() != nil {
				return
			}
			b.checkDoneWatch(ctx, watch)
		}
		wait := doneWatchPollInterval
		for _, watch := range b.store.DoneWatches() {
			if until := watch.NextAt.Sub(b.now()); until > 0 && until < wait {
				wait = until
			}
		}
		if err := b.sleep(ctx, wait); err != nil {
			return
		}
	}
}

func (b *Bot) checkDoneWatch(ctx context.Context, watch state.DoneWatch) {
	key := watch.Key()
	if !b.userAllowed(watch.AuthorID) || !b.conversationAllowed(watch.Channel) {
		log.Printf("slackbot: drop done watch %s: user %q or channel is no longer allowed", key, watch.AuthorID)
		b.endDoneWatch(ctx, watch, "")
		return
	}
	reaction := watchReaction(watch)
	reacted, err := b.api.ReactionUsers(ctx, watch.Channel, watch.MessageTS, reaction)
	if err != nil {
		if response, ok := errors.AsType[slack.SlackErrorResponse](err); ok && response.Err == "message_not_found" {
			b.endDoneWatch(ctx, watch, b.text().doneWatchGoneMessage)
			return
		}
		log.Printf("slackbot: read reactions of %s: %v", key, err)
		return
	}
	pending := pendingUsers(watch.Targets, reacted)
	if len(pending) == 0 {
		b.endDoneWatch(ctx, watch, withReaction(b.text().doneWatchCompleted, reaction))
		return
	}
	// A crash after the last reminder was counted leaves the watch behind.
	if watch.Reminders >= maxDoneWatchReminders {
		b.endDoneWatch(ctx, watch, "")
		return
	}
	if b.now().Before(watch.NextAt) {
		return
	}
	// Count the reminder before posting, so a crash cannot remind twice, and
	// only while the watch is still saved, so a stop during this check wins.
	now := b.now()
	hour := b.settings().doneWatchHour()
	watch, ok, err := b.store.UpdateDoneWatch(key, func(saved *state.DoneWatch) bool {
		if now.Before(saved.NextAt) {
			return false
		}
		saved.Reminders++
		saved.NextAt = nextDoneWatchReminder(now, saved.IntervalDays, hour)
		return true
	})
	if err != nil {
		log.Printf("slackbot: save done watch %s: %v", key, err)
		return
	}
	if !ok {
		return
	}
	original := b.text().doneWatchOriginal
	if link, err := b.api.Permalink(ctx, watch.Channel, watch.MessageTS); err == nil {
		original = "<" + link + "|" + b.text().doneWatchOriginal + ">"
	} else {
		log.Printf("slackbot: link done watch message %s: %v", key, err)
	}
	text := withReaction(fmt.Sprintf(b.text().doneWatchReminder, userMentions(pending), original, watch.Reminders), reaction)
	last := watch.Reminders >= maxDoneWatchReminders
	if last {
		text += fmt.Sprintf(b.text().doneWatchLastReminder, maxDoneWatchReminders)
	}
	if err := b.post(ctx, watch.Channel, watch.ThreadTS, text); err != nil {
		log.Printf("slackbot: post done watch reminder %s: %v", key, err)
	}
	if last {
		b.endDoneWatch(ctx, watch, "")
	}
}

func (b *Bot) endDoneWatch(ctx context.Context, watch state.DoneWatch, message string) {
	key := watch.Key()
	removed, err := b.store.DeleteDoneWatches(func(w state.DoneWatch) bool { return w.Key() == key })
	if err != nil {
		log.Printf("slackbot: delete done watch %s: %v", key, err)
		return
	}
	// A stop during this check already removed the watch and said so.
	if len(removed) == 0 || message == "" {
		return
	}
	if err := b.post(ctx, watch.Channel, watch.ThreadTS, message); err != nil {
		log.Printf("slackbot: post done watch end %s: %v", key, err)
	}
}

func (b *Bot) stopDoneWatches(channel, threadTS string) bool {
	removed, err := b.store.DeleteDoneWatches(func(w state.DoneWatch) bool {
		return w.Channel == channel && w.ThreadTS == threadTS
	})
	if err != nil {
		log.Printf("slackbot: stop done watches in %s: %v", threadRef(channel, threadTS), err)
	}
	return len(removed) > 0
}

func doneWatchSummary(text *messages, watch state.DoneWatch) string {
	return fmt.Sprintf(text.doneWatchSummary, len(watch.Targets), doneWatchIntervalLabel(text, watch.IntervalDays, watch.NextAt.In(time.Local).Hour()), formatFollowUpTime(watch.NextAt))
}

func (b *Bot) stopDoneWatchCommand(ctx context.Context, channel, threadTS string) {
	message := b.text().doneWatchNothingToStop
	if b.stopDoneWatches(channel, threadTS) {
		message = b.text().doneWatchStoppedMessage
	}
	b.postDoneWatchNotice(ctx, channel, threadTS, message)
}
