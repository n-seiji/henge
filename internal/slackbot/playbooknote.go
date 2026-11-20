package slackbot

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/n-seiji/ebi-x/internal/codex"
	"github.com/n-seiji/ebi-x/internal/playbook"
	"github.com/n-seiji/ebi-x/internal/prompt"
	"github.com/n-seiji/ebi-x/internal/workspace"
)

const (
	// Notes are meant to bridge edits, not to replace them, so the lesson ends
	// up in the playbook itself, where every later turn follows it.
	playbookFoldThreshold = 3
	maxFoldSummaryRunes   = 300
)

type playbookFold struct {
	name    string
	notes   int
	changed bool
	summary string
}

// Beside the memory, which the agent can neither read nor write directly.
func (b *Bot) playbookNotesDir() string {
	return filepath.Join(b.config.MemoryDir, "playbooks")
}

func (b *Bot) currentPlaybooks() ([]playbook.Playbook, error) {
	playbooks := append([]playbook.Playbook(nil), b.playbooks...)
	if b.config.PlaybooksDir != "" {
		var err error
		if playbooks, err = playbook.List(b.config.PlaybooksDir); err != nil {
			return nil, err
		}
	}
	if b.config.MemoryDir != "" {
		b.memoryMu.RLock()
		err := playbook.AttachNotes(b.playbookNotesDir(), playbooks)
		b.memoryMu.RUnlock()
		if err != nil {
			log.Printf("slackbot: read playbook notes: %v", err)
		}
	}
	return playbooks, nil
}

// Playbooks reach every channel, so only channels that may change them may add
// notes, as with global memory.
func (b *Bot) savePlaybookNote(eventKey string, note *codex.PlaybookNote, invalid, sharedWritable bool) (notice string, fold *playbook.Playbook) {
	if invalid {
		log.Printf("slackbot: ignore malformed playbook note output %q", eventKey)
		return b.text().playbookNoteInvalidNotice, nil
	}
	if note == nil {
		return "", nil
	}
	if !sharedWritable || b.config.MemoryDir == "" {
		log.Printf("slackbot: ignore playbook note from channel without shared writes %q", eventKey)
		return "", nil
	}
	playbooks, err := b.currentPlaybooks()
	if err != nil {
		log.Printf("slackbot: list playbooks for note %q: %v", eventKey, err)
		return fmt.Sprintf(b.text().playbookNoteFailed, note.Target), nil
	}
	target, ok := playbook.Find(playbooks, note.Target)
	if !ok {
		return fmt.Sprintf(b.text().playbookNoteMissing, note.Target), nil
	}
	b.memoryMu.Lock()
	written, err := playbook.AppendNote(b.playbookNotesDir(), target, note.Body, b.now())
	b.memoryMu.Unlock()
	if err != nil {
		log.Printf("slackbot: append playbook note %q: %v", eventKey, err)
		return fmt.Sprintf(b.text().playbookNoteFailed, target.Name), nil
	}
	if written == "" {
		return "", nil
	}
	notice = fmt.Sprintf(b.text().playbookNoteSaved, target.Name, written)
	if current, ok := b.findPlaybook(target.Name); ok && noteCount(current) >= playbookFoldThreshold {
		fold = &current
	}
	return notice, fold
}

func (b *Bot) findPlaybook(name string) (playbook.Playbook, bool) {
	playbooks, err := b.currentPlaybooks()
	if err != nil {
		log.Printf("slackbot: reload playbook %q: %v", name, err)
		return playbook.Playbook{}, false
	}
	return playbook.Find(playbooks, name)
}

func noteCount(item playbook.Playbook) int {
	if item.Notes == "" {
		return 0
	}
	return len(strings.Split(item.Notes, "\n"))
}

// Folds without asking, as the operator chose.
func (b *Bot) foldPlaybookNotes(ctx context.Context, sessionID string, lease *workspace.Lease, scope channelScope, item playbook.Playbook, onActivity func(codex.Activity)) *playbookFold {
	fold := &playbookFold{name: item.Name, notes: noteCount(item)}
	text, err := b.runWorkTurn(ctx, sessionID, lease, scope, true, prompt.BuildPlaybookFoldPrompt(item), func(string) error { return nil }, onActivity)
	if err != nil {
		log.Printf("slackbot: fold notes of playbook %q: %v", item.Name, err)
		return fold
	}
	fold.summary = clipText(strings.Join(strings.Fields(codex.SanitizeSlackOutput(text)), " "), maxFoldSummaryRunes)
	if after, ok := b.findPlaybook(item.Name); ok && !after.ModTime.Equal(item.ModTime) {
		fold.changed = true
	}
	return fold
}

func (f *playbookFold) notice(text *messages) string {
	switch {
	case f.changed:
		return fmt.Sprintf(text.playbookFoldedNotice, f.notes, f.name, f.summary)
	case f.summary != "":
		return fmt.Sprintf(text.playbookFoldSkipped, f.name, f.notes, f.summary)
	default:
		return fmt.Sprintf(text.playbookFoldFailed, f.name)
	}
}

func (b *Bot) recordPlaybookFold(ctx context.Context, fold *playbookFold, channel, threadTS string) {
	if fold == nil || !fold.changed || b.config.ApprovalChannelID == "" {
		return
	}
	link := b.threadLink(ctx, b.text(), make(map[string]string), channel, threadTS)
	if err := b.post(ctx, b.config.ApprovalChannelID, "", fmt.Sprintf(b.text().playbookFoldAdminPost, fold.name, fold.notes, link, fold.summary)); err != nil {
		log.Printf("slackbot: record playbook fold %q: %v", fold.name, err)
	}
}
