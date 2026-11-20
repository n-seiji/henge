package slackbot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/n-seiji/ebi-x/internal/github"
	"github.com/n-seiji/ebi-x/internal/state"
)

const (
	pullWatchPollInterval = 2 * time.Minute
	pullWatchTTL          = 7 * 24 * time.Hour
	maxPullWatchWakes     = 10
	maxPullWatchesPerTurn = 5
	maxPullActivityItems  = 10
	maxPullActivityRunes  = 300
)

type GitHub interface {
	ParsePullURL(raw string) (github.PullRef, bool)
	Pull(ctx context.Context, ref github.PullRef) (github.Pull, error)
	Checks(ctx context.Context, ref github.PullRef, sha string) (github.Checks, error)
	ActivitySince(ctx context.Context, ref github.PullRef, cursor github.Cursor, ignoreAuthor string) ([]github.Activity, github.Cursor, error)
}

func (b *Bot) pullWatchEnabled() bool {
	return b.config.GitHub != nil
}

// A watch starts from the pull request's current state, so only later CI
// results and comments wake the session.
func (b *Bot) watchPulls(ctx context.Context, trigger processingTrigger, urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	if !b.pullWatchEnabled() {
		return b.text().pullWatchDisabledNotice
	}
	var started, invalid, foreign, taken, failed, closed []string
	tooMany := false
	for i, raw := range urls {
		if i >= maxPullWatchesPerTurn {
			tooMany = true
			break
		}
		ref, ok := b.config.GitHub.ParsePullURL(raw)
		if !ok {
			invalid = append(invalid, shortDetail(raw))
			continue
		}
		if !b.pullRepoWatchable(ref) {
			foreign = append(foreign, ref.String())
			continue
		}
		// One watch per pull request: another thread's watch is not moved
		// here, so its events keep reaching the thread that started it.
		if previous, ok := b.store.GetPullWatch(ref.String()); ok &&
			(previous.Channel != trigger.channel || previous.ThreadTS != trigger.threadTS) {
			taken = append(taken, ref.String())
			continue
		}
		pull, err := b.config.GitHub.Pull(ctx, ref)
		if err != nil {
			log.Printf("slackbot: read pull request %s: %v", ref, err)
			failed = append(failed, ref.String())
			continue
		}
		if pull.State != "open" {
			closed = append(closed, pull.URL)
			continue
		}
		checks, err := b.config.GitHub.Checks(ctx, ref, pull.HeadSHA)
		if err != nil {
			log.Printf("slackbot: read checks of %s: %v", ref, err)
			failed = append(failed, ref.String())
			continue
		}
		_, cursor, err := b.config.GitHub.ActivitySince(ctx, ref, github.Cursor{}, b.config.GitHubLogin)
		if err != nil {
			log.Printf("slackbot: read activity of %s: %v", ref, err)
			failed = append(failed, ref.String())
			continue
		}
		now := b.now()
		watch := state.PullWatch{
			Channel: trigger.channel, ThreadTS: trigger.threadTS, AuthorID: trigger.authorID,
			Owner: ref.Owner, Repo: ref.Repo, Number: ref.Number, URL: pull.URL,
			HeadSHA: pull.HeadSHA, Checks: checks.State, Cursor: state.PullCursor(cursor),
			CreatedAt: now, ExpiresAt: now.Add(pullWatchTTL),
		}
		// A turn the watch itself started keeps counting toward the cap;
		// a person's request in the thread starts the count again.
		if previous, ok := b.store.GetPullWatch(watch.Key()); ok && trigger.source == pullWatchTrigger {
			watch.Wakes = previous.Wakes
			watch.CreatedAt = previous.CreatedAt
		}
		if err := b.store.SetPullWatch(watch); err != nil {
			log.Printf("slackbot: save pull request watch %s: %v", ref, err)
			failed = append(failed, ref.String())
			continue
		}
		started = append(started, pull.URL)
	}
	var notices []string
	if len(started) > 0 {
		notices = append(notices, b.text().pullWatchStartedNotice+bulletList(started))
	}
	if len(invalid) > 0 {
		notices = append(notices, b.text().pullWatchInvalidNotice+bulletList(invalid))
	}
	if len(foreign) > 0 {
		notices = append(notices, b.text().pullWatchRepoNotice+bulletList(foreign))
	}
	if len(taken) > 0 {
		notices = append(notices, b.text().pullWatchTakenNotice+bulletList(taken))
	}
	if len(failed) > 0 {
		notices = append(notices, b.text().pullWatchFailedNotice+bulletList(failed))
	}
	if len(closed) > 0 {
		notices = append(notices, b.text().pullWatchClosedNotice+bulletList(closed))
	}
	if tooMany {
		notices = append(notices, fmt.Sprintf(b.text().pullWatchTooManyNotice, maxPullWatchesPerTurn))
	}
	return strings.Join(notices, "\n\n")
}

func (b *Bot) pullRepoWatchable(ref github.PullRef) bool {
	name := ref.Owner + "/" + ref.Repo
	for _, repo := range b.config.GitHubRepos {
		if strings.EqualFold(repo, name) {
			return true
		}
	}
	return false
}

func bulletList(items []string) string {
	return "\n- " + strings.Join(items, "\n- ")
}

func (b *Bot) runPullWatches(acceptCtx, turnCtx context.Context, wg *sync.WaitGroup) {
	if !b.pullWatchEnabled() {
		return
	}
	for {
		for _, watch := range b.store.PullWatches() {
			if acceptCtx.Err() != nil {
				return
			}
			b.checkPullWatch(acceptCtx, turnCtx, wg, watch)
		}
		if err := b.sleep(acceptCtx, pullWatchPollInterval); err != nil {
			return
		}
	}
}

func (b *Bot) checkPullWatch(ctx, turnCtx context.Context, wg *sync.WaitGroup, watch state.PullWatch) {
	key := watch.Key()
	// A turn already working in the thread may be pushing to the pull
	// request; look again once it is done.
	if b.threadBusy(threadRef(watch.Channel, watch.ThreadTS)) {
		return
	}
	if !b.userAllowed(watch.AuthorID) || !b.conversationAllowed(watch.Channel) {
		log.Printf("slackbot: drop pull request watch %s: user %q or channel is no longer allowed", key, watch.AuthorID)
		b.endPullWatch(ctx, watch, "")
		return
	}
	if !b.now().Before(watch.ExpiresAt) {
		b.endPullWatch(ctx, watch, fmt.Sprintf(b.text().pullWatchExpiredMessage, watch.URL))
		return
	}
	ref := github.PullRef{Owner: watch.Owner, Repo: watch.Repo, Number: watch.Number}
	pull, err := b.config.GitHub.Pull(ctx, ref)
	if err != nil {
		log.Printf("slackbot: read pull request %s: %v", key, err)
		return
	}
	switch {
	case pull.Merged:
		b.endPullWatch(ctx, watch, fmt.Sprintf(b.text().pullMergedMessage, watch.URL))
		return
	case pull.State != "open":
		b.endPullWatch(ctx, watch, fmt.Sprintf(b.text().pullClosedMessage, watch.URL))
		return
	}
	checks, err := b.config.GitHub.Checks(ctx, ref, pull.HeadSHA)
	if err != nil {
		log.Printf("slackbot: read checks of %s: %v", key, err)
		return
	}
	activity, cursor, err := b.config.GitHub.ActivitySince(ctx, ref, github.Cursor(watch.Cursor), b.config.GitHubLogin)
	if err != nil {
		log.Printf("slackbot: read activity of %s: %v", key, err)
		return
	}

	previousChecks := watch.Checks
	if pull.HeadSHA != watch.HeadSHA {
		// A new commit's checks start over; its earlier failure was handled.
		previousChecks = ""
	}
	failed := checks.State == github.ChecksFailure && previousChecks != github.ChecksFailure
	events := pullEvents(&english, failed, checks, activity)
	passed := checks.State == github.ChecksSuccess && watch.Checks == github.ChecksFailure
	changed := pull.HeadSHA != watch.HeadSHA || checks.State != watch.Checks || github.Cursor(watch.Cursor) != cursor
	watch.HeadSHA = pull.HeadSHA
	watch.Checks = checks.State
	watch.Cursor = state.PullCursor(cursor)

	if passed {
		if err := b.post(ctx, watch.Channel, watch.ThreadTS, fmt.Sprintf(b.text().pullChecksPassedMessage, watch.URL)); err != nil {
			log.Printf("slackbot: post checks passed %s: %v", key, err)
		}
	}
	if len(events) == 0 {
		if changed {
			if err := b.store.SetPullWatch(watch); err != nil {
				log.Printf("slackbot: save pull request watch %s: %v", key, err)
			}
		}
		return
	}
	if watch.Wakes >= maxPullWatchWakes {
		b.endPullWatch(ctx, watch, fmt.Sprintf(b.text().pullWatchLimitMessage, maxPullWatchWakes, watch.URL))
		return
	}
	watch.Wakes++
	watch.ExpiresAt = b.now().Add(pullWatchTTL)
	// Save before the turn starts, so a crash or the next poll cannot
	// report the same events twice.
	if err := b.store.SetPullWatch(watch); err != nil {
		log.Printf("slackbot: save pull request watch %s: %v", key, err)
		return
	}
	summary := strings.Join(pullEvents(b.text(), failed, checks, activity), b.text().pullEventSeparator)
	task := describePullEvent(pull, checks, activity, events)
	wg.Go(func() { b.firePullWatch(turnCtx, watch, summary, task) })
}

func (b *Bot) firePullWatch(ctx context.Context, watch state.PullWatch, summary, task string) {
	var timestamp string
	if err := b.retrySlack(ctx, func() error {
		var err error
		timestamp, err = b.api.PostMessage(ctx, watch.Channel, watch.ThreadTS, fmt.Sprintf(b.text().pullWatchWakeMessage, watch.URL, summary))
		return err
	}); err != nil {
		log.Printf("slackbot: announce pull request event %s: %v", watch.Key(), err)
		return
	}
	b.processTrigger(ctx, processingTrigger{
		source:      pullWatchTrigger,
		authorID:    watch.AuthorID,
		channel:     watch.Channel,
		timestamp:   timestamp,
		threadTS:    watch.ThreadTS,
		message:     task,
		threadReply: true,
	})
}

// The session reads the English events; people read the configured language.
func pullEvents(text *messages, failed bool, checks github.Checks, activity []github.Activity) []string {
	var events []string
	if failed {
		events = append(events, fmt.Sprintf(text.pullEventCIFailed, strings.Join(checks.Failed, text.pullEventSeparator)))
	}
	if len(activity) > 0 {
		events = append(events, fmt.Sprintf(text.pullEventActivity, len(activity)))
	}
	return events
}

// Bodies are clipped; the session can read them in full on GitHub.
func describePullEvent(pull github.Pull, checks github.Checks, activity []github.Activity, events []string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "PR: %s (%s)\n", pull.URL, pull.Title)
	for _, event := range events {
		builder.WriteString("- " + event + "\n")
	}
	for i, item := range activity {
		if i >= maxPullActivityItems {
			fmt.Fprintf(&builder, "- %d more\n", len(activity)-i)
			break
		}
		label := map[string]string{
			github.KindReview:        "review",
			github.KindReviewComment: "code comment",
			github.KindComment:       "comment",
		}[item.Kind]
		if item.State != "" && item.Kind == github.KindReview {
			label += " (" + item.State + ")"
		}
		if item.Path != "" {
			label += " " + item.Path
		}
		fmt.Fprintf(&builder, "- %[2]s by @%[1]s: %[3]s %[4]s\n", item.Author, label, clipText(item.Body, maxPullActivityRunes), item.URL)
	}
	return strings.TrimSpace(builder.String())
}

func (b *Bot) endPullWatch(ctx context.Context, watch state.PullWatch, message string) {
	key := watch.Key()
	if _, err := b.store.DeletePullWatches(func(w state.PullWatch) bool { return w.Key() == key }); err != nil {
		log.Printf("slackbot: delete pull request watch %s: %v", key, err)
		return
	}
	if message == "" {
		return
	}
	if err := b.post(ctx, watch.Channel, watch.ThreadTS, message); err != nil {
		log.Printf("slackbot: post pull request watch end %s: %v", key, err)
	}
}

func (b *Bot) stopPullWatches(channel, threadTS string) bool {
	removed, err := b.store.DeletePullWatches(func(w state.PullWatch) bool {
		return w.Channel == channel && w.ThreadTS == threadTS
	})
	if err != nil {
		log.Printf("slackbot: stop pull request watches in %s: %v", threadRef(channel, threadTS), err)
	}
	return len(removed) > 0
}

func (b *Bot) threadBusy(threadKey string) bool {
	b.runningMu.Lock()
	defer b.runningMu.Unlock()
	return len(b.running[threadKey]) > 0
}
