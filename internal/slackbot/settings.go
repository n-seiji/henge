package slackbot

import (
	"fmt"
	"log"
	"regexp"
)

const (
	defaultDoneReaction  = "done"
	defaultDoneWatchHour = 19
)

var reactionName = regexp.MustCompile(`^[a-z0-9_+'-]+$`)

type Settings struct {
	DoneWatch struct {
		Reaction string `yaml:"reaction"`
		Hour     *int   `yaml:"hour"`
	} `yaml:"done_watch"`
	SQLWriteServers []string `yaml:"sql_write_servers"`
	Language        string   `yaml:"language"`
}

func ReadSettings(path string) (Settings, error) {
	var settings Settings
	if err := readYAMLFile(path, "settings", &settings); err != nil {
		return Settings{}, err
	}
	if reaction := settings.DoneWatch.Reaction; reaction != "" && !reactionName.MatchString(reaction) {
		return Settings{}, fmt.Errorf("settings %q: done_watch.reaction %q is not an emoji name without colons", path, reaction)
	}
	if hour := settings.DoneWatch.Hour; hour != nil && (*hour < 0 || *hour > 23) {
		return Settings{}, fmt.Errorf("settings %q: done_watch.hour %d is not between 0 and 23", path, *hour)
	}
	if language := settings.Language; language != "" && language != "ja" && language != "en" {
		return Settings{}, fmt.Errorf("settings %q: language %q is not ja or en", path, language)
	}
	return settings, nil
}

// A broken file falls back to defaults rather than stopping work, because
// every default is the safe side: no SQL writes and the built-in reaction.
func (b *Bot) settings() Settings {
	settings, err := ReadSettings(b.config.SettingsFile)
	if err != nil {
		log.Printf("slackbot: %v", err)
	}
	return settings
}

func (b *Bot) settingsNotice() string {
	if _, err := ReadSettings(b.config.SettingsFile); err != nil {
		return fmt.Sprintf(b.text().settingsBrokenNotice, err)
	}
	return ""
}

func (s Settings) doneReaction() string {
	if s.DoneWatch.Reaction == "" {
		return defaultDoneReaction
	}
	return s.DoneWatch.Reaction
}

func (s Settings) doneWatchHour() int {
	if s.DoneWatch.Hour == nil {
		return defaultDoneWatchHour
	}
	return *s.DoneWatch.Hour
}

func (b *Bot) text() *messages {
	if b.settings().Language == "en" {
		return &english
	}
	return &japanese
}
