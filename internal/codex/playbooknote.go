package codex

const (
	playbookNoteHeading   = "## Playbook note"
	playbookNoteTargetKey = "Playbook"
	playbookNoteBodyKey   = "Text"
)

// Target is kept as the model wrote it; the bot decides whether it names a
// playbook.
type PlaybookNote struct {
	Target string
	Body   string
}

func SplitPlaybookNote(text string) (rest string, note *PlaybookNote, invalid bool) {
	rest, fields, present, invalid := splitFieldSection(text, playbookNoteHeading, playbookNoteTargetKey, playbookNoteBodyKey)
	if !present {
		return text, nil, false
	}
	if invalid || fields[playbookNoteTargetKey] == "" || fields[playbookNoteBodyKey] == "" {
		return rest, nil, true
	}
	return rest, &PlaybookNote{Target: fields[playbookNoteTargetKey], Body: fields[playbookNoteBodyKey]}, false
}
