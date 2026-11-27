package slackbot

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/n-seiji/henge/internal/codex"
	"github.com/n-seiji/henge/internal/state"
)

const (
	usageCheckInterval = 30 * time.Minute
	usageHistoryKeep   = 12 * time.Hour
	usageReadTimeout   = 30 * time.Second
	usageDayLayout     = "2006-01-02"
	// Used when the reset time could not be read, so recordUsage cannot
	// announce the limit.
	adminAlertInterval = time.Hour
)

var usageThresholds = []int{10, 0}

var usageQuestionWords = []string{"usage", "残り", "利用枠", "使用量"}

type UsageReader interface {
	ReadRateLimits(ctx context.Context) (*codex.RateLimits, error)
}

func (b *Bot) usageWatchEnabled() bool {
	return b.config.Usage != nil && b.config.ApprovalChannelID != ""
}

func (b *Bot) runUsageWatch(ctx context.Context) {
	if !b.usageWatchEnabled() {
		return
	}
	failing := false
	for {
		limits, err := b.readUsage(ctx)
		switch {
		case err != nil && !failing:
			// Logged once per outage; an API-key login never has usage.
			log.Printf("slackbot: read Codex usage: %v", err)
			failing = true
		case err == nil:
			failing = false
			b.recordUsage(ctx, limits)
			b.reportUsage(ctx)
		}
		if err := b.sleep(ctx, usageCheckInterval); err != nil {
			return
		}
	}
}

func (b *Bot) readUsage(ctx context.Context) (*codex.RateLimits, error) {
	readCtx, cancel := context.WithTimeout(ctx, usageReadTimeout)
	defer cancel()
	return b.config.Usage.ReadRateLimits(readCtx)
}

func isUsageQuestion(text string) bool {
	lower := strings.ToLower(text)
	for _, word := range usageQuestionWords {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

func (b *Bot) usageReply(ctx context.Context) string {
	text := b.text()
	if b.config.Usage == nil {
		return text.usageUnavailableReply
	}
	limits, err := b.readUsage(ctx)
	if err != nil {
		log.Printf("slackbot: read Codex usage for a question: %v", err)
		return text.usageUnavailableReply
	}
	windows := usageWindows(limits)
	var parts []string
	for _, key := range usageWindowKeys {
		window, ok := windows[key]
		if !ok {
			continue
		}
		part := fmt.Sprintf("%s %d%%", usageWindowLabel(text, key, window), usageRemaining(window))
		if at := window.ResetTime(); !at.IsZero() {
			part += fmt.Sprintf(text.usageReplyReset, formatFollowUpTime(at))
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return text.usageUnavailableReply
	}
	return text.usageReplyPrefix + strings.Join(parts, text.listSeparator)
}

func (b *Bot) recordUsage(ctx context.Context, limits *codex.RateLimits) {
	b.usageMu.Lock()
	defer b.usageMu.Unlock()
	saved := b.store.Usage()
	next := saved
	quiet := isWeekend(b.now())
	day := b.now().In(time.Local).Format(usageDayLayout)
	if saved.Day != day {
		if summary := usageDailySummary(b.text(), saved); summary != "" && !quiet {
			if err := b.post(ctx, b.config.ApprovalChannelID, "", summary); err != nil {
				log.Printf("slackbot: post Codex usage summary: %v", err)
			}
		}
		next.Day = day
		next.Windows = nil
	}
	windows := usageWindows(limits)
	days := make(map[string]state.UsageDay, len(windows))
	for key, record := range next.Windows {
		days[key] = record
	}
	for key, window := range windows {
		remaining := math.Max(0, 100-window.UsedPercent)
		record, ok := days[key]
		if !ok {
			record = state.UsageDay{First: remaining, Min: remaining}
		}
		record.Label = usageWindowLabel(b.text(), key, window)
		record.Last = remaining
		record.Min = math.Min(record.Min, remaining)
		record.Checks++
		days[key] = record
	}
	next.Windows = days
	next.History = appendUsageSample(next.History, usageSample(b.now(), windows))

	alerts, lines := usageAlerts(b.text(), saved.Alerts, windows)
	next.Alerts = alerts
	if quiet {
		// Announce it at the first weekday check instead.
		next.Alerts = saved.Alerts
	} else if len(lines) > 0 {
		text := strings.Join(lines, "\n") + "\n" + b.text().usageSummaryPrefix + usageSummary(b.text(), limits)
		if err := b.post(ctx, b.config.ApprovalChannelID, "", text); err != nil {
			log.Printf("slackbot: post Codex usage alert: %v", err)
			// Announce it again at the next check.
			next.Alerts = saved.Alerts
		}
	}
	if err := b.store.SetUsage(next); err != nil {
		log.Printf("slackbot: save Codex usage: %v", err)
	}
}

func (b *Bot) reportUsage(ctx context.Context) {
	b.usageMu.Lock()
	defer b.usageMu.Unlock()
	saved := b.store.Usage()
	if len(saved.History) == 0 {
		return
	}
	now := b.now()
	if isWeekend(now) {
		return
	}
	previous := saved.Reported
	if previous != nil && sameLocalHour(previous.At, now) {
		return
	}
	latest := saved.History[len(saved.History)-1]
	if now.Sub(latest.At) >= usageCheckInterval {
		// This check was not recorded; do not report an older one as now.
		return
	}
	key, ok := usageReportedKey(latest)
	if !ok {
		return
	}
	if previous != nil && !usageSampleChanged(*previous, latest, key) {
		// Compared with the last report, not the last check, so a change
		// seen by a check that was not posted is not lost.
		return
	}
	text := usageHourlyReport(b.text(), latest, key)
	if err := b.post(ctx, b.config.ApprovalChannelID, "", text); err != nil {
		log.Printf("slackbot: post Codex usage report: %v", err)
		// Report again at the next check.
		return
	}
	next := saved
	next.Reported = &latest
	if err := b.store.SetUsage(next); err != nil {
		log.Printf("slackbot: save Codex usage: %v", err)
	}
}

// The 5-hour window refills within the day, so only the longest window is
// worth an hourly post.
func usageReportedKey(sample state.UsageSample) (string, bool) {
	best, bestMinutes := "", int64(0)
	for _, key := range usageWindowKeys {
		if _, ok := sample.Remaining[key]; !ok {
			continue
		}
		if minutes := usageSampleMinutes(sample, key); best == "" || minutes > bestMinutes {
			best, bestMinutes = key, minutes
		}
	}
	return best, best != ""
}

func usageHourlyReport(text *messages, latest state.UsageSample, key string) string {
	line := fmt.Sprintf(text.usageHourlyLine, latest.Remaining[key])
	if resetsAt := latest.ResetsAt[key]; resetsAt > 0 {
		line += fmt.Sprintf(text.usageNotes, fmt.Sprintf(text.usageHourlyResetAt, formatFollowUpTime(time.Unix(resetsAt, 0))))
	}
	return line
}

// Falls back to Codex's usual 5-hour and weekly windows for checks that did not
// keep the length.
func usageSampleMinutes(sample state.UsageSample, key string) int64 {
	switch minutes := sample.Minutes[key]; {
	case minutes > 0:
		return minutes
	case key == "primary":
		return 5 * 60
	default:
		return 7 * 24 * 60
	}
}

func usageSample(at time.Time, windows map[string]codex.RateLimitWindow) state.UsageSample {
	sample := state.UsageSample{At: at, Remaining: make(map[string]float64, len(windows)), ResetsAt: make(map[string]int64, len(windows)), Minutes: make(map[string]int64, len(windows))}
	for key, window := range windows {
		sample.Remaining[key] = math.Max(0, 100-window.UsedPercent)
		if window.ResetsAt > 0 {
			sample.ResetsAt[key] = window.ResetsAt
		}
		if window.WindowMinutes > 0 {
			sample.Minutes[key] = window.WindowMinutes
		}
	}
	return sample
}

func appendUsageSample(history []state.UsageSample, sample state.UsageSample) []state.UsageSample {
	history = append(history, sample)
	cutoff := sample.At.Add(-usageHistoryKeep)
	first := 0
	for first < len(history) && history[first].At.Before(cutoff) {
		first++
	}
	return history[first:]
}

func usageSampleChanged(a, b state.UsageSample, key string) bool {
	remaining, ok := a.Remaining[key]
	other, otherOK := b.Remaining[key]
	return ok != otherOK || math.Round(remaining*10) != math.Round(other*10) || a.ResetsAt[key] != b.ResetsAt[key]
}

func sameLocalHour(a, b time.Time) bool {
	return localHour(a).Equal(localHour(b))
}

// Not time.Truncate: it counts from UTC, which splits hours wrongly in zones
// offset by 30 minutes.
func localHour(t time.Time) time.Time {
	t = t.In(time.Local)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.Local)
}

func usageAlerts(text *messages, announced map[string]int, windows map[string]codex.RateLimitWindow) (map[string]int, []string) {
	next := make(map[string]int, len(windows))
	var lines []string
	for _, key := range usageWindowKeys {
		window, ok := windows[key]
		if !ok {
			continue
		}
		level, crossed := usageLevel(window)
		if !crossed {
			// Back above every threshold: the window was reset.
			continue
		}
		next[key] = level
		if previous, ok := announced[key]; ok && previous <= level {
			// Already announced, or risen since without a full reset.
			continue
		}
		label := usageWindowLabel(text, key, window)
		if level == 0 {
			lines = append(lines, fmt.Sprintf(text.usageExhaustedMessage, label, formatUsageReset(text, window)))
			continue
		}
		reset := ""
		if at := window.ResetTime(); !at.IsZero() {
			reset = fmt.Sprintf(text.usageResetSuffix, formatFollowUpTime(at))
		}
		lines = append(lines, fmt.Sprintf(text.usageLowMessage, label, level, reset))
	}
	return next, lines
}

func usageDailySummary(text *messages, usage state.Usage) string {
	day, err := time.ParseInLocation(usageDayLayout, usage.Day, time.Local)
	if err != nil {
		return ""
	}
	record, ok := usage.Windows[usageDailyKey(usage)]
	if !ok {
		return ""
	}
	return fmt.Sprintf(text.usageDailyLine, day.Format("1/2"), int(math.Floor(record.First)), int(math.Floor(record.Last)), int(math.Floor(record.Min)))
}

// The day's records keep no window length, so the latest check decides, as
// it does for the hourly post. Without one, the secondary slot is the longer
// window on plans that have two.
func usageDailyKey(usage state.Usage) string {
	if n := len(usage.History); n > 0 {
		if key, ok := usageReportedKey(usage.History[n-1]); ok {
			if _, kept := usage.Windows[key]; kept {
				return key
			}
		}
	}
	best := ""
	for _, key := range usageWindowKeys {
		if _, ok := usage.Windows[key]; ok {
			best = key
		}
	}
	return best
}

func (b *Bot) usageLimitFailure(ctx context.Context) string {
	retry := b.text().usageLimitRetryLater
	var limits *codex.RateLimits
	if b.config.Usage != nil {
		var err error
		if limits, err = b.readUsage(ctx); err != nil {
			log.Printf("slackbot: read Codex usage after the usage limit: %v", err)
			limits = nil
		}
	}
	if limits != nil {
		var latest time.Time
		for _, window := range limits.Windows() {
			if at := window.ResetTime(); window.UsedPercent >= 100 && at.After(latest) {
				latest = at
			}
		}
		if !latest.IsZero() {
			retry = fmt.Sprintf(b.text().usageLimitRetryAt, formatFollowUpTime(latest))
		}
	}
	notice := b.text().adminContactText
	if b.config.ApprovalChannelID != "" {
		notice = b.text().adminNotifiedText
		if limits != nil {
			b.recordUsage(ctx, limits)
		} else {
			b.alertAdmin(ctx, "usage", b.text().usageLimitAdminAlert)
		}
	}
	return fmt.Sprintf(b.text().usageLimitFailureMessage, notice, retry)
}

func (b *Bot) authFailure(ctx context.Context) string {
	if b.config.ApprovalChannelID == "" {
		return fmt.Sprintf(b.text().authFailureMessage, b.text().adminContactText)
	}
	b.alertAdmin(ctx, "auth", b.text().authAdminAlert)
	return fmt.Sprintf(b.text().authFailureMessage, b.text().adminNotifiedText)
}

func (b *Bot) alertAdmin(ctx context.Context, kind, text string) {
	b.usageMu.Lock()
	defer b.usageMu.Unlock()
	now := b.now()
	if last, ok := b.adminAlerts[kind]; ok && now.Sub(last) < adminAlertInterval {
		return
	}
	if err := b.post(ctx, b.config.ApprovalChannelID, "", text); err != nil {
		log.Printf("slackbot: post %s alert: %v", kind, err)
		return
	}
	if b.adminAlerts == nil {
		b.adminAlerts = make(map[string]time.Time)
	}
	b.adminAlerts[kind] = now
}

var usageWindowKeys = []string{"primary", "secondary"}

// Keyed by slot, which stays the same across resets.
func usageWindows(limits *codex.RateLimits) map[string]codex.RateLimitWindow {
	windows := make(map[string]codex.RateLimitWindow, 2)
	if limits.Primary != nil {
		windows["primary"] = *limits.Primary
	}
	if limits.Secondary != nil {
		windows["secondary"] = *limits.Secondary
	}
	return windows
}

func usageLevel(window codex.RateLimitWindow) (int, bool) {
	remaining := 100 - window.UsedPercent
	level, crossed := 0, false
	for _, threshold := range usageThresholds {
		if threshold == 0 {
			if window.UsedPercent >= 100 {
				level, crossed = 0, true
			}
			continue
		}
		if remaining < float64(threshold) {
			level, crossed = threshold, true
		}
	}
	return level, crossed
}

func usageRemaining(window codex.RateLimitWindow) int {
	return int(math.Max(0, math.Floor(100-window.UsedPercent)))
}

func usageWindowLabel(text *messages, key string, window codex.RateLimitWindow) string {
	minutes := window.WindowMinutes
	switch {
	case minutes == 7*24*60:
		return text.durationWeek
	case minutes > 0 && minutes%(24*60) == 0:
		return fmt.Sprintf(text.durationDays, minutes/(24*60))
	case minutes > 0 && minutes%60 == 0:
		return fmt.Sprintf(text.durationHours, minutes/60)
	case minutes > 0:
		return fmt.Sprintf(text.durationMinutes, minutes)
	case key == "primary":
		return fmt.Sprintf(text.durationHours, 5)
	default:
		return text.durationWeek
	}
}

func formatUsageReset(text *messages, window codex.RateLimitWindow) string {
	if at := window.ResetTime(); !at.IsZero() {
		return fmt.Sprintf(text.usageResetsAt, formatFollowUpTime(at))
	}
	return text.usageUnknownReset
}

func usageSummary(text *messages, limits *codex.RateLimits) string {
	var parts []string
	windows := usageWindows(limits)
	for _, key := range usageWindowKeys {
		window, ok := windows[key]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d%%", usageWindowLabel(text, key, window), usageRemaining(window)))
	}
	return strings.Join(parts, text.listSeparator)
}
