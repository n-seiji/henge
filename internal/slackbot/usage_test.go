package slackbot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/n-seiji/henge/internal/codex"
	"github.com/n-seiji/henge/internal/state"
)

type fakeUsage struct {
	limits *codex.RateLimits
	err    error
	calls  int
}

func (u *fakeUsage) ReadRateLimits(context.Context) (*codex.RateLimits, error) {
	u.calls++
	return u.limits, u.err
}

func usageLimits(primaryUsed, secondaryUsed float64) *codex.RateLimits {
	return &codex.RateLimits{
		Primary:   &codex.RateLimitWindow{UsedPercent: primaryUsed, WindowMinutes: 300, ResetsAt: time.Date(2026, 10, 20, 6, 30, 0, 0, time.UTC).Unix()},
		Secondary: &codex.RateLimitWindow{UsedPercent: secondaryUsed, WindowMinutes: 7 * 24 * 60, ResetsAt: time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC).Unix()},
	}
}

func newUsageBot(t *testing.T, store *fakeStore, api *fakeSlack, usage UsageReader) *Bot {
	t.Helper()
	bot := newTestBot(t, store, api, &fakeRunner{})
	bot.config.ApprovalChannelID = "CAPPROVAL"
	bot.config.Usage = usage
	// A Tuesday, since nothing is posted on weekends.
	bot.now = func() time.Time { return time.Date(2026, 10, 20, 10, 0, 0, 0, time.Local) }
	return bot
}

func TestRecordUsageAnnouncesOncePerLevel(t *testing.T) {
	api := &fakeSlack{}
	store := &fakeStore{}
	bot := newUsageBot(t, store, api, &fakeUsage{})
	ctx := context.Background()

	steps := []struct {
		primary, secondary float64
		want               []string
	}{
		{primary: 50, secondary: 10},
		{primary: 85, secondary: 75},
		{primary: 91, secondary: 10, want: []string{"Codex の利用枠（5時間）の残りが 10% を下回りました。"}},
		{primary: 95, secondary: 10},
		{primary: 100, secondary: 92, want: []string{"Codex の利用枠（5時間）を使い切りました。", "まで、henge は作業できません。", "（1週間）の残りが 10% を下回りました"}},
		{primary: 100, secondary: 93},
		// The 5-hour window resets; a later fall is announced again.
		{primary: 5, secondary: 93},
		{primary: 92, secondary: 93, want: []string{"（5時間）の残りが 10% を下回りました"}},
	}
	for i, step := range steps {
		before := len(api.posts)
		bot.recordUsage(ctx, usageLimits(step.primary, step.secondary))
		posted := api.posts[before:]
		if len(step.want) == 0 {
			if len(posted) != 0 {
				t.Fatalf("step %d: posted %q, want nothing", i, posted[0].text)
			}
			continue
		}
		if len(posted) != 1 {
			t.Fatalf("step %d: posts = %d, want 1", i, len(posted))
		}
		if posted[0].channel != "CAPPROVAL" || posted[0].threadTS != "" {
			t.Fatalf("step %d: posted to %q/%q", i, posted[0].channel, posted[0].threadTS)
		}
		for _, want := range step.want {
			if !strings.Contains(posted[0].text, want) {
				t.Fatalf("step %d: post %q does not contain %q", i, posted[0].text, want)
			}
		}
		if !strings.Contains(posted[0].text, japanese.usageSummaryPrefix) {
			t.Fatalf("step %d: post %q has no summary", i, posted[0].text)
		}
	}
}

func TestRecordUsageAlertSurvivesRestart(t *testing.T) {
	api := &fakeSlack{}
	store := &fakeStore{}
	newUsageBot(t, store, api, &fakeUsage{}).recordUsage(context.Background(), usageLimits(95, 0))
	newUsageBot(t, store, api, &fakeUsage{}).recordUsage(context.Background(), usageLimits(96, 0))
	if len(api.posts) != 1 {
		t.Fatalf("posts = %d, want 1", len(api.posts))
	}
}

func TestRecordUsagePostsDailySummary(t *testing.T) {
	api := &fakeSlack{}
	store := &fakeStore{}
	bot := newUsageBot(t, store, api, &fakeUsage{})
	now := time.Date(2026, 10, 20, 9, 0, 0, 0, time.Local)
	bot.now = func() time.Time { return now }
	ctx := context.Background()

	for _, used := range [][2]float64{{10, 20}, {70, 30}, {30, 45}} {
		bot.recordUsage(ctx, usageLimits(used[0], used[1]))
		now = now.Add(30 * time.Minute)
	}
	if len(api.posts) != 0 {
		t.Fatalf("posted %q before the day ended", api.posts[0].text)
	}
	now = time.Date(2026, 10, 21, 0, 10, 0, 0, time.Local)
	bot.recordUsage(ctx, usageLimits(0, 46))

	if len(api.posts) != 1 || api.posts[0].channel != "CAPPROVAL" {
		t.Fatalf("posts = %+v, want one summary", api.posts)
	}
	want := "📊 10/20 の Codex の利用枠（3回確認）\n- 5時間: 90% → 70%（最低 30%）\n- 1週間: 80% → 55%（最低 55%）"
	if api.posts[0].text != want {
		t.Fatalf("summary = %q, want %q", api.posts[0].text, want)
	}
	if got := store.Usage(); got.Day != "2026-10-21" || got.Windows["primary"].Checks != 1 {
		t.Fatalf("usage after the summary = %+v", got)
	}
}

func TestUsageWatchPostsNothingOnWeekends(t *testing.T) {
	api := &fakeSlack{}
	store := &fakeStore{}
	bot := newUsageBot(t, store, api, &fakeUsage{})
	ctx := context.Background()
	now := time.Date(2026, 10, 23, 18, 5, 0, 0, time.Local)
	bot.now = func() time.Time { return now }
	check := func(primary, secondary float64) {
		bot.recordUsage(ctx, usageLimits(primary, secondary))
		bot.reportUsage(ctx)
	}
	check(20, 10)
	if len(api.posts) != 1 {
		t.Fatalf("posts on Friday = %d, want the hourly report", len(api.posts))
	}

	// Saturday: the day's change, a fall below 10% and new hours are all
	// recorded but not posted.
	for _, at := range []time.Time{
		time.Date(2026, 10, 24, 0, 5, 0, 0, time.Local),
		time.Date(2026, 10, 24, 1, 5, 0, 0, time.Local),
		time.Date(2026, 10, 25, 23, 35, 0, 0, time.Local),
	} {
		now = at
		check(95, 50)
	}
	if len(api.posts) != 1 {
		t.Fatalf("posted %q on the weekend, want nothing", api.posts[1].text)
	}
	if got := store.usage; got.Day != "2026-10-25" || got.Windows["primary"].Checks != 1 {
		t.Fatalf("usage after the weekend = %+v, want Sunday's check recorded", got)
	}

	// Monday's first check announces the window still below 10%.
	now = time.Date(2026, 10, 26, 0, 5, 0, 0, time.Local)
	check(96, 50)
	var texts []string
	for _, post := range api.posts[1:] {
		texts = append(texts, post.text)
	}
	joined := strings.Join(texts, "\n")
	for _, want := range []string{"📊 10/25 の Codex の利用枠", "（5時間）の残りが 10% を下回りました", "残：50.0%"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("Monday posts %q do not contain %q", texts, want)
		}
	}
}

func TestApprovalChannelAnswersUsageQuestion(t *testing.T) {
	bot, api, _, _ := newApprovalBot(t, &fakeRunner{})
	bot.config.Usage = &fakeUsage{limits: usageLimits(72.4, 37)}

	reply := approvalCommand(bot, api, "U1", "", "残りどれくらい？")

	if !strings.HasPrefix(reply, japanese.usageReplyPrefix+"5時間 27%（") || !strings.Contains(reply, "・1週間 63%（") {
		t.Fatalf("reply = %q", reply)
	}
}

func TestApprovalChannelUsageQuestionWithoutUsage(t *testing.T) {
	bot, api, _, _ := newApprovalBot(t, &fakeRunner{})
	bot.config.Usage = &fakeUsage{err: errors.New("api key")}

	if reply := approvalCommand(bot, api, "UOWNER", "", "usage"); reply != japanese.usageUnavailableReply {
		t.Fatalf("reply = %q", reply)
	}
}

func TestUsageLevel(t *testing.T) {
	tests := []struct {
		used    float64
		level   int
		crossed bool
	}{
		{used: 90, crossed: false},
		{used: 90.5, level: 10, crossed: true},
		{used: 99.9, level: 10, crossed: true},
		{used: 100, level: 0, crossed: true},
	}
	for _, tt := range tests {
		level, crossed := usageLevel(codex.RateLimitWindow{UsedPercent: tt.used})
		if level != tt.level || crossed != tt.crossed {
			t.Fatalf("usageLevel(%v) = %d, %v; want %d, %v", tt.used, level, crossed, tt.level, tt.crossed)
		}
	}
}

func TestUsageLimitFailureNamesReset(t *testing.T) {
	api := &fakeSlack{}
	limits := usageLimits(100, 40)
	bot := newUsageBot(t, &fakeStore{}, api, &fakeUsage{limits: limits})
	message := bot.usageLimitFailure(context.Background())
	if !strings.Contains(message, japanese.adminNotifiedText+formatFollowUpTime(limits.Primary.ResetTime())+"以降に") {
		t.Fatalf("message = %q", message)
	}
	if len(api.posts) != 1 || api.posts[0].channel != "CAPPROVAL" {
		t.Fatalf("posts = %+v, want one announcement", api.posts)
	}
}

func TestUsageLimitFailureWithoutUsage(t *testing.T) {
	api := &fakeSlack{}
	bot := newUsageBot(t, &fakeStore{}, api, &fakeUsage{err: errors.New("api key")})
	for range 2 {
		if message := bot.usageLimitFailure(context.Background()); !strings.Contains(message, japanese.adminNotifiedText+japanese.usageLimitRetryLater) {
			t.Fatalf("message = %q", message)
		}
	}
	// The second failure within the hour does not repeat the alert.
	if len(api.posts) != 1 || api.posts[0].channel != "CAPPROVAL" || api.posts[0].text != japanese.usageLimitAdminAlert {
		t.Fatalf("posts = %+v, want one alert", api.posts)
	}
}

func TestUsageLimitFailureWithoutApprovalChannel(t *testing.T) {
	api := &fakeSlack{}
	bot := newTestBot(t, &fakeStore{}, api, &fakeRunner{})
	if message := bot.usageLimitFailure(context.Background()); !strings.Contains(message, japanese.adminContactText) {
		t.Fatalf("message = %q", message)
	}
	if len(api.posts) != 0 {
		t.Fatalf("posts = %+v, want none", api.posts)
	}
}

func TestWorkTurnNotSignedInAlertsAdmin(t *testing.T) {
	for _, response := range []runnerResponse{
		{result: &codex.TurnResult{ThreadID: "thread-1", Err: "Your access token could not be refreshed because your refresh token has expired. Please log out and sign in again."}},
		{err: fmt.Errorf("codex exec: unexpected status 401 Unauthorized: %w", codex.ErrNotSignedIn)},
	} {
		api := &fakeSlack{}
		runner := &fakeRunner{responses: []runnerResponse{response}}
		bot := newTestBot(t, &fakeStore{claim: true}, api, runner)
		bot.config.ApprovalChannelID = "CAPPROVAL"
		now := time.Date(2026, 10, 20, 9, 0, 0, 0, time.Local)
		bot.now = func() time.Time { return now }

		bot.HandleMention(context.Background(), mention())

		var alerts, replies []string
		for _, post := range api.posts {
			if post.channel == "CAPPROVAL" {
				alerts = append(alerts, post.text)
			} else {
				replies = append(replies, post.text)
			}
		}
		if len(alerts) != 1 || alerts[0] != japanese.authAdminAlert {
			t.Fatalf("alerts = %q", alerts)
		}
		if want := fmt.Sprintf(japanese.authFailureMessage, japanese.adminNotifiedText); len(replies) != 1 || replies[0] != want {
			t.Fatalf("replies = %q, want %q", replies, want)
		}
	}
}

func TestWorkTurnUsageLimitExplainsFailure(t *testing.T) {
	api := &fakeSlack{}
	runner := &fakeRunner{responses: []runnerResponse{{result: &codex.TurnResult{
		ThreadID: "thread-1",
		Err:      "You’ve hit your usage limit. Try again at 3:30 PM.",
	}}}}
	store := &fakeStore{claim: true}
	bot := newTestBot(t, store, api, runner)

	bot.HandleMention(context.Background(), mention())

	got := strings.Join(api.postTexts, "\n")
	if !strings.Contains(got, "Codex の利用上限に達したため") || strings.Contains(got, japanese.workFailureMessage) {
		t.Fatalf("posts = %q", api.postTexts)
	}
}

func TestReportUsagePostsWeeklyLineHourly(t *testing.T) {
	api := &fakeSlack{}
	store := &fakeStore{}
	bot := newUsageBot(t, store, api, &fakeUsage{})
	ctx := context.Background()
	now := time.Date(2026, 10, 20, 1, 5, 0, 0, time.Local)
	bot.now = func() time.Time { return now }
	reset := "（Reset at " + formatFollowUpTime(time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)) + "）"

	check := func(primary, secondary float64) {
		bot.recordUsage(ctx, usageLimits(primary, secondary))
		bot.reportUsage(ctx)
	}
	check(20, 10)
	if len(api.posts) != 1 {
		t.Fatalf("posts = %d, want the first report", len(api.posts))
	}
	first := api.posts[0]
	if first.channel != "CAPPROVAL" || first.threadTS != "" {
		t.Fatalf("posted to %q/%q", first.channel, first.threadTS)
	}
	if want := "残：90.0%" + reset; first.text != want {
		t.Fatalf("report = %q, want %q", first.text, want)
	}

	// Later checks in the same hour are recorded but not reported.
	now = now.Add(usageCheckInterval)
	check(30, 11)
	if len(api.posts) != 1 {
		t.Fatalf("posts = %d, want no report within the hour", len(api.posts))
	}

	// An hour where only the 5-hour window moved is skipped.
	now = time.Date(2026, 10, 20, 2, 5, 0, 0, time.Local)
	check(60, 10)
	if len(api.posts) != 1 {
		t.Fatalf("posts = %d, want an hour without a weekly change skipped", len(api.posts))
	}

	// The weekly window moving again is reported after the quiet hour.
	now = now.Add(time.Hour)
	check(70, 12.5)
	if len(api.posts) != 2 {
		t.Fatalf("posts = %d, want a report after the quiet hour", len(api.posts))
	}
	if want := "残：87.5%" + reset; api.posts[1].text != want {
		t.Fatalf("report = %q, want %q", api.posts[1].text, want)
	}
}

func TestReportUsageReportsAWeeklyPrimaryWindow(t *testing.T) {
	api := &fakeSlack{}
	store := &fakeStore{}
	bot := newUsageBot(t, store, api, &fakeUsage{})
	bot.now = func() time.Time { return time.Date(2026, 10, 20, 1, 5, 0, 0, time.Local) }
	// A plan whose only window, in the primary slot, is weekly.
	weekly := &codex.RateLimits{Primary: &codex.RateLimitWindow{UsedPercent: 18, WindowMinutes: 7 * 24 * 60}}
	bot.recordUsage(context.Background(), weekly)
	bot.reportUsage(context.Background())
	if len(api.posts) != 1 {
		t.Fatalf("posts = %d, want 1", len(api.posts))
	}
	if text := api.posts[0].text; text != "残：82.0%" {
		t.Fatalf("report %q does not show the weekly window", text)
	}
}

func TestReportUsageSkipsUnrecordedCheck(t *testing.T) {
	api := &fakeSlack{}
	store := &fakeStore{}
	bot := newUsageBot(t, store, api, &fakeUsage{})
	at := time.Date(2026, 10, 20, 1, 5, 0, 0, time.Local)
	store.usage.History = []state.UsageSample{{At: at, Remaining: map[string]float64{"primary": 80}}}
	bot.now = func() time.Time { return at.Add(time.Hour) }
	bot.reportUsage(context.Background())
	if len(api.posts) != 0 {
		t.Fatalf("posted %q, want an old check not reported", api.posts[0].text)
	}
}

func TestAppendUsageSampleDropsOldChecks(t *testing.T) {
	at := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	var history []state.UsageSample
	for i := range 200 {
		history = appendUsageSample(history, state.UsageSample{At: at.Add(time.Duration(i) * usageCheckInterval)})
	}
	if first := history[0].At; history[len(history)-1].At.Sub(first) > usageHistoryKeep {
		t.Fatalf("history spans from %v, longer than %v", first, usageHistoryKeep)
	}
}
