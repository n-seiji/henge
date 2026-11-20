package slackbot

import (
	"fmt"
	"log"

	"github.com/n-seiji/ebi-x/internal/usersettings"
)

const ()

func (b *Bot) readUserSettings(userID string) []string {
	if b.config.MemoryDir == "" || !usersettings.ValidUserID(userID) {
		return nil
	}
	b.memoryMu.RLock()
	settings, err := usersettings.Read(b.config.MemoryDir, userID)
	b.memoryMu.RUnlock()
	if err != nil {
		log.Printf("slackbot: read user settings %q: %v", userID, err)
	}
	return settings
}

// Only a request a person wrote changes settings, and only that person's, so a
// follow-up, a pull request event, or a scheduled run cannot change anyone's
// settings.
func (b *Bot) saveUserSettings(eventKey string, trigger processingTrigger, settings []string) string {
	if trigger.source != mentionTrigger && trigger.source != messageTrigger {
		log.Printf("slackbot: ignore user settings from a request nobody wrote %q", eventKey)
		return ""
	}
	if b.config.MemoryDir == "" || !usersettings.ValidUserID(trigger.authorID) {
		return ""
	}
	b.memoryMu.Lock()
	written, err := usersettings.Write(b.config.MemoryDir, trigger.authorID, settings)
	b.memoryMu.Unlock()
	if err != nil {
		log.Printf("slackbot: write user settings %q: %v", eventKey, err)
		return b.text().userSettingsFailedNotice
	}
	if len(written) == 0 {
		return b.text().userSettingsClearedNotice
	}
	return fmt.Sprintf(b.text().userSettingsSavedNotice, len(written))
}
