package slackbot

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestReadSettingsRejectsMistakes(t *testing.T) {
	for _, content := range []string{
		"done_watch:\n  emoji: done\n",
		"done_watch:\n  reaction: \":done:\"\n",
		"done_watch:\n  hour: 24\n",
		"sql_write_servers: ledger\n",
		"language: fr\n",
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

func TestEnglishSettingRepliesInEnglish(t *testing.T) {
	store := &fakeStore{claim: true}
	api := &fakeSlack{reactionUsers: []string{"U3"}}
	bot := newFollowUpBot(t, store, api, &fakeRunner{})
	bot.config.SettingsFile = writeChannelRules(t, "language: en\n")

	bot.HandleMention(context.Background(), doneWatchMention("<@U2> <@U3> please check <@UBOT> watch this"))

	if post := api.postTexts[len(api.postTexts)-1]; !strings.Contains(post, "Every weekday at 19:00") {
		t.Fatalf("notice = %q, want it in English", post)
	}
}

// A translation that drops or reorders a verb prints %!d(string=...) to Slack,
// which go vet cannot see through a struct field.
func TestEnglishMessagesTakeTheSameArguments(t *testing.T) {
	verb := regexp.MustCompile(`%(?:\[(\d)\])?[-+# 0-9.]*[a-zA-Z%]`)
	arguments := func(format string) []string {
		var args []string
		next := 1
		for _, match := range verb.FindAllStringSubmatch(format, -1) {
			if strings.HasSuffix(match[0], "%") {
				continue
			}
			index := next
			if match[1] != "" {
				index = int(match[1][0] - '0')
			}
			args = append(args, fmt.Sprintf("%d%c", index, match[0][len(match[0])-1]))
			next = index + 1
		}
		slices.Sort(args)
		return slices.Compact(args)
	}
	ja, en := reflect.ValueOf(japanese), reflect.ValueOf(english)
	for i := range ja.NumField() {
		if ja.Field(i).Kind() != reflect.String {
			continue
		}
		if got, want := arguments(en.Field(i).String()), arguments(ja.Field(i).String()); !slices.Equal(got, want) {
			t.Errorf("%s takes %v in English, %v in Japanese", ja.Type().Field(i).Name, got, want)
		}
	}
}

func TestMessagesCallTheBotByItsSlackName(t *testing.T) {
	store := &fakeStore{}
	bot := newTestBot(t, store, &fakeSlack{}, &fakeRunner{})
	bot.setBotName("helper")

	for _, language := range []string{"ja", "en"} {
		bot.config.SettingsFile = writeChannelRules(t, "language: "+language+"\n")
		text := reflect.ValueOf(*bot.text())
		for i := range text.NumField() {
			if field := text.Field(i); field.Kind() == reflect.String && strings.Contains(field.String(), defaultBotName) {
				t.Errorf("%s %s still says %q: %q", language, text.Type().Field(i).Name, defaultBotName, field.String())
			}
		}
		if !strings.Contains(bot.text().forbiddenMessage, "helper") {
			t.Errorf("%s forbidden message = %q, want the Slack name", language, bot.text().forbiddenMessage)
		}
	}
}
