package codex

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	scheduleHeading  = "## Schedule"
	ScheduleStopWord = "stop"
)

var weekdayNames = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

var (
	everyDay = [7]bool{true, true, true, true, true, true, true}
	weekdays = [7]bool{false, true, true, true, true, true, false}
	weekends = [7]bool{true, false, false, false, false, false, true}
)

// Running at most daily keeps unattended work bounded.
type ScheduleSpec struct {
	Days   [7]bool
	Hour   int
	Minute int
}

func ParseScheduleSpec(when string) (ScheduleSpec, error) {
	fields := strings.Fields(strings.ReplaceAll(strings.TrimSpace(when), "：", ":"))
	if len(fields) != 2 {
		return ScheduleSpec{}, errors.New("want days and a time")
	}
	var spec ScheduleSpec
	var err error
	if spec.Days, err = parseScheduleDays(fields[0]); err != nil {
		return ScheduleSpec{}, err
	}
	hour, minute, ok := strings.Cut(fields[1], ":")
	if !ok {
		return ScheduleSpec{}, errors.New("time is not HH:MM")
	}
	if spec.Hour, err = strconv.Atoi(hour); err != nil || spec.Hour < 0 || spec.Hour > 23 {
		return ScheduleSpec{}, fmt.Errorf("bad hour %q", hour)
	}
	if spec.Minute, err = strconv.Atoi(minute); err != nil || len(minute) != 2 || spec.Minute < 0 || spec.Minute > 59 {
		return ScheduleSpec{}, fmt.Errorf("bad minute %q", minute)
	}
	return spec, nil
}

func parseScheduleDays(days string) ([7]bool, error) {
	switch strings.ToLower(days) {
	case "daily":
		return everyDay, nil
	case "weekdays":
		return weekdays, nil
	case "weekends":
		return weekends, nil
	}
	if names, ok := strings.CutPrefix(days, "毎週"); ok {
		return parseLegacyWeekdays(names)
	}
	switch days {
	case "毎日":
		return everyDay, nil
	case "平日":
		return weekdays, nil
	case "週末":
		return weekends, nil
	}
	var result [7]bool
	for _, name := range strings.Split(days, ",") {
		day := slices.IndexFunc(weekdayNames, func(weekday string) bool { return strings.EqualFold(weekday, name) })
		if day < 0 {
			return [7]bool{}, fmt.Errorf("unknown days %q", days)
		}
		result[day] = true
	}
	return result, nil
}

// Schedules saved before the spec was in English are stored in this form and
// are parsed again on every run.
func parseLegacyWeekdays(names string) ([7]bool, error) {
	names = strings.TrimSuffix(strings.TrimSuffix(names, "日"), "曜")
	names = strings.NewReplacer("・", "", "、", "", ",", "", "曜", "").Replace(names)
	if names == "" {
		return [7]bool{}, errors.New("no weekday")
	}
	var result [7]bool
	for _, name := range names {
		day := strings.IndexRune("日月火水木金土", name)
		if day < 0 {
			return [7]bool{}, fmt.Errorf("unknown weekday %q", string(name))
		}
		result[utf8.RuneCountInString("日月火水木金土"[:day])] = true
	}
	return result, nil
}

func (s ScheduleSpec) String() string {
	var days string
	switch s.Days {
	case everyDay:
		days = "daily"
	case weekdays:
		days = "weekdays"
	case weekends:
		days = "weekends"
	default:
		var names []string
		// Monday first, as people list a week.
		for _, i := range []int{1, 2, 3, 4, 5, 6, 0} {
			if s.Days[i] {
				names = append(names, weekdayNames[i])
			}
		}
		days = strings.Join(names, ",")
	}
	return fmt.Sprintf("%s %02d:%02d", days, s.Hour, s.Minute)
}

func (s ScheduleSpec) Next(after time.Time) time.Time {
	day := time.Date(after.Year(), after.Month(), after.Day(), 0, 0, 0, 0, after.Location())
	for i := 0; i <= 7; i++ {
		date := day.AddDate(0, 0, i)
		at := time.Date(date.Year(), date.Month(), date.Day(), s.Hour, s.Minute, 0, 0, after.Location())
		if s.Days[at.Weekday()] && at.After(after) {
			return at
		}
	}
	// Unreachable for a spec with a day: the loop covers a whole week.
	return time.Time{}
}

type ScheduleRequest struct {
	When string
	Task string
	Stop bool
}

func SplitSchedule(text string) (rest string, request *ScheduleRequest, invalid bool) {
	rest, fields, present, invalid := splitFieldSection(text, scheduleHeading, followUpWhenKey, followUpTaskKey)
	if !present {
		return text, nil, false
	}
	if invalid {
		return rest, nil, true
	}
	if strings.EqualFold(contractWord(fields[followUpWhenKey]), ScheduleStopWord) {
		return rest, &ScheduleRequest{Stop: true}, false
	}
	if fields[followUpWhenKey] == "" || fields[followUpTaskKey] == "" {
		return rest, nil, true
	}
	return rest, &ScheduleRequest{When: fields[followUpWhenKey], Task: fields[followUpTaskKey]}, false
}
