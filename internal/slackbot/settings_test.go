package slackbot

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestReadSettingsRejectsMistakes(t *testing.T) {
	for _, content := range []string{
		"done_watch:\n  emoji: done\n",
		"done_watch:\n  reaction: \":done:\"\n",
		"done_watch:\n  hour: 24\n",
		"sql_write_servers: ledger\n",
	} {
		if _, err := ReadSettings(writeChannelRules(t, content)); err == nil {
			t.Errorf("ReadSettings(%q) accepted it", content)
		}
	}
}

func TestDoneWatchFollowsSettings(t *testing.T) {
	store := &fakeStore{claim: true}
	api := &fakeSlack{reactionUsers: []string{"U3"}}
	bot := newFollowUpBot(t, store, api, &fakeRunner{})
	bot.config.SettingsFile = writeChannelRules(t, "done_watch:\n  reaction: ok_hand\n  hour: 9\n")

	bot.HandleMention(context.Background(), doneWatchMention("<@U2> <@U3> 確認お願いします <@UBOT> 監視して"))

	got := store.doneWatches["C1:100.1"]
	if got.Reaction != "ok_hand" || api.reactionUsersName != "ok_hand" || got.NextAt.Hour() != 9 {
		t.Fatalf("watch = %+v, reaction read %q; want ok_hand at 9", got, api.reactionUsersName)
	}
	if post := api.postTexts[len(api.postTexts)-1]; !strings.Contains(post, "平日の毎日9時") || !strings.Contains(post, ":ok_hand:") || strings.Contains(post, ":done:") {
		t.Fatalf("notice = %q", post)
	}
}

func TestWorkTurnGetsSQLWriteServers(t *testing.T) {
	runner := completedRunner()
	bot := newTestBot(t, &fakeStore{claim: true}, &fakeSlack{}, runner)
	bot.config.SettingsFile = writeChannelRules(t, "sql_write_servers: [ledger]\n")

	bot.HandleMention(context.Background(), mention())

	if len(runner.access) != 1 || !reflect.DeepEqual(runner.access[0].SQLWriters, []string{"ledger"}) {
		t.Fatalf("access = %+v, want ledger allowed to write", runner.access)
	}
}
