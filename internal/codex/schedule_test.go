package codex

import (
	"testing"
	"time"
)

func TestParseScheduleSpec(t *testing.T) {
	for input, want := range map[string]string{
		"daily 09:00":       "daily 09:00",
		"Weekdays 9:30":     "weekdays 09:30",
		"weekends 10：00":    "weekends 10:00",
		"Mon,Wed,Fri 08:15": "Mon,Wed,Fri 08:15",
		"thu,mon 9:00":      "Mon,Thu 09:00",
		"Sun 23:59":         "Sun 23:59",
		"  Tue   7:05 ":     "Tue 07:05",
		"毎日 09:00":          "daily 09:00",
		"平日 9:30":           "weekdays 09:30",
		"週末 10:00":          "weekends 10:00",
		"毎週月水金曜 08:15":      "Mon,Wed,Fri 08:15",
		"毎週月・木曜日 9:00":      "Mon,Thu 09:00",
		"毎週日曜 23:59":        "Sun 23:59",
	} {
		spec, err := ParseScheduleSpec(input)
		if err != nil {
			t.Errorf("ParseScheduleSpec(%q) error = %v", input, err)
			continue
		}
		if got := spec.String(); got != want {
			t.Errorf("ParseScheduleSpec(%q) = %q, want %q", input, got, want)
		}
		if again, err := ParseScheduleSpec(spec.String()); err != nil || again != spec {
			t.Errorf("String() of %q does not read back: %v", input, err)
		}
	}
	for _, input := range []string{"", "daily", "hourly 00", "30m", "daily 24:00", "daily 9:5", "Mon, Wed 9:00", "Mon,X 9:00", "daily 9:00 from", "毎週 9:00", "毎週月X 9:00"} {
		if _, err := ParseScheduleSpec(input); err == nil {
			t.Errorf("ParseScheduleSpec(%q) accepted", input)
		}
	}
}

func TestScheduleSpecNext(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	// 2026-10-09 is a Friday.
	friday := time.Date(2026, 10, 9, 9, 0, 0, 0, jst)
	weekdays, _ := ParseScheduleSpec("weekdays 09:00")
	if got, want := weekdays.Next(friday.Add(-time.Minute)), friday; !got.Equal(want) {
		t.Errorf("before the time: %v, want %v", got, want)
	}
	if got, want := weekdays.Next(friday), time.Date(2026, 10, 12, 9, 0, 0, 0, jst); !got.Equal(want) {
		t.Errorf("at the time: %v, want next Monday %v", got, want)
	}
	sunday, _ := ParseScheduleSpec("Sun 08:00")
	if got, want := sunday.Next(time.Date(2026, 10, 11, 8, 0, 0, 0, jst)), time.Date(2026, 10, 18, 8, 0, 0, 0, jst); !got.Equal(want) {
		t.Errorf("weekly: %v, want a week later %v", got, want)
	}
}

func TestSplitSchedule(t *testing.T) {
	rest, request, invalid := SplitSchedule("設定します。\n\n## Schedule\n- When: weekdays 09:00\n- Task: CI の失敗を確認する\n\n以上")
	if invalid || request == nil || request.When != "weekdays 09:00" || request.Task != "CI の失敗を確認する" || request.Stop {
		t.Fatalf("SplitSchedule() = %#v, %v", request, invalid)
	}
	if rest != "設定します。\n\n以上" {
		t.Fatalf("rest = %q", rest)
	}
	if _, request, invalid := SplitSchedule("止めます。\n## Schedule\n- When: stop"); invalid || request == nil || !request.Stop {
		t.Fatalf("stop = %#v, %v", request, invalid)
	}
	if _, request, invalid := SplitSchedule("## Schedule\n- When: weekdays 09:00"); !invalid || request != nil {
		t.Fatalf("missing task = %#v, %v", request, invalid)
	}
	if rest, request, invalid := SplitSchedule("本文\n```\n## Schedule\n```"); invalid || request != nil || rest != "本文\n```\n## Schedule\n```" {
		t.Fatalf("fenced heading = %q, %#v, %v", rest, request, invalid)
	}
}
