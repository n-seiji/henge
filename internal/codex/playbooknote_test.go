package codex

import "testing"

func TestSplitPlaybookNote(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantRest    string
		wantNote    *PlaybookNote
		wantInvalid bool
	}{
		{
			name:     "no section",
			text:     "回答です。",
			wantRest: "回答です。",
		},
		{
			name:     "note",
			text:     "回答です。\n\n## Playbook note\n- Playbook: slides\n- Text: 納品は3週後から2週後に変わった\n\n## Channel memory\n用語",
			wantRest: "回答です。\n\n## Channel memory\n用語",
			wantNote: &PlaybookNote{Target: "slides", Body: "納品は3週後から2週後に変わった"},
		},
		{
			name:     "full-width colon and backquoted name",
			text:     "## Playbook note\n- Playbook：`slides`\n- Text：締切は金曜",
			wantNote: &PlaybookNote{Target: "slides", Body: "締切は金曜"},
		},
		{
			name:        "missing body",
			text:        "回答です。\n## Playbook note\n- Playbook: slides",
			wantRest:    "回答です。",
			wantInvalid: true,
		},
		{
			name:        "repeated section",
			text:        "回答です。\n## Playbook note\n- Playbook: a\n- Text: x\n## Playbook note\n- Playbook: b\n- Text: y",
			wantRest:    "回答です。",
			wantInvalid: true,
		},
		{
			name:     "heading inside a code block is prose",
			text:     "```\n## Playbook note\n- Playbook: a\n```",
			wantRest: "```\n## Playbook note\n- Playbook: a\n```",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rest, note, invalid := SplitPlaybookNote(tt.text)
			if rest != tt.wantRest || invalid != tt.wantInvalid {
				t.Errorf("SplitPlaybookNote() rest = %q, invalid = %v; want %q, %v", rest, invalid, tt.wantRest, tt.wantInvalid)
			}
			if (note == nil) != (tt.wantNote == nil) || (note != nil && *note != *tt.wantNote) {
				t.Errorf("SplitPlaybookNote() note = %#v, want %#v", note, tt.wantNote)
			}
		})
	}
}
