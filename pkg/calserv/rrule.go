package calserv

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/nais/gemini/pkg/azure"
)

type RecurringRule struct {
	rules map[string]string
}

func (r *RecurringRule) Serialize() string {
	buf := &bytes.Buffer{}

	write := func(k, v string) {
		buf.WriteString(k)
		buf.WriteRune('=')
		buf.WriteString(v)
		buf.WriteRune(';')
	}

	// first, write the required FREQ.
	// either UNTIL or COUNT may appear in a 'recur',
	// but UNTIL and COUNT MUST NOT occur in the same 'recur'
	initRules := []string{
		RuleFrequency,
		RuleUntil,
		RuleCount,
		RuleInterval,
		RuleBySecond,
		RuleByMinute,
		RuleByHour,
		RuleByDay,
		RuleByMonthDay,
		RuleByYearDay,
		RuleByWeekNo,
		RuleByMonth,
		RuleBySetPos,
		RuleWeekStart,
	}

	for _, k := range initRules {
		v := r.rules[k]
		delete(r.rules, k)
		if len(v) == 0 {
			continue
		}
		write(k, v)
	}

	// the rest of these keywords are optional,
	// but MUST NOT occur more than once
	for k, v := range r.rules {
		write(k, v)
	}

	s := buf.String()
	return s[:len(s)-1]
}

func (r *RecurringRule) Add(k, v string) {
	k = strings.ToUpper(k)
	v = strings.ToUpper(v)
	r.rules[k] = v
}

func utctime(t time.Time) string {
	const format = "20060102T150405Z"
	//return t.Format(format)
	return t.Local().Format(format)
}

// https://www.ietf.org/rfc/rfc2445.txt
// 4.3.10 Recurrence Rule
//                 ( ";" "INTERVAL" "=" 1*DIGIT )          /
//                ( ";" "BYSECOND" "=" byseclist )        /
//                ( ";" "BYMINUTE" "=" byminlist )        /
//                ( ";" "BYHOUR" "=" byhrlist )           /
//                ( ";" "BYDAY" "=" bywdaylist )          /
//                ( ";" "BYMONTHDAY" "=" bymodaylist )    /
//                ( ";" "BYYEARDAY" "=" byyrdaylist )     /
//                ( ";" "BYWEEKNO" "=" bywknolist )       /
//                ( ";" "BYMONTH" "=" bymolist )          /
//                ( ";" "BYSETPOS" "=" bysplist )         /
//                ( ";" "WKST" "=" weekday )              /
const (
	RuleFrequency  = "FREQ"
	RuleUntil      = "UNTIL"
	RuleCount      = "COUNT"
	RuleInterval   = "INTERVAL"
	RuleBySecond   = "BYSECOND"
	RuleByMinute   = "BYMINUTE"
	RuleByHour     = "BYHOUR"
	RuleByDay      = "BYDAY"
	RuleByMonthDay = "BYMONTHDAY"
	RuleByYearDay  = "BYYEARDAY"
	RuleByWeekNo   = "BYWEEKNO"
	RuleByMonth    = "BYMONTH"
	RuleBySetPos   = "BYSETPOS"
	RuleWeekStart  = "WKST"
)

func daysOfWeek(days []string) string {
	buf := &bytes.Buffer{}
	for _, day := range days {
		if len(day) < 2 {
			continue
		}
		buf.WriteString(strings.ToUpper(day[:2]))
		buf.WriteRune(',')
	}
	s := buf.String()
	if len(s) == 0 {
		return s
	}
	return s[:len(s)-1]
}

var weekIndex = map[string]string{
	"first":  "1",
	"second": "2",
	"third":  "3",
	"fourth": "4",
	"last":   "-1",
}

func setRecurrence(event azure.Event, e *ics.VEvent) {
	if event.Recurrence == nil {
		return
	}
	r := event.Recurrence

	rule := &RecurringRule{
		rules: map[string]string{},
	}

	rule.Add(RuleFrequency, r.Pattern.Type)
	rule.Add(RuleByDay, weekIndex[r.Pattern.Index]+daysOfWeek(r.Pattern.DaysOfWeek))

	switch strings.ToUpper(r.Pattern.Type) {
	case "SECONDLY", "MINUTELY", "HOURLY", "DAILY", "MONTHLY", "YEARLY":
	// built-in types according to RFC
	case "WEEKLY":
		// built-in types according to RFC
		rule.Add(RuleByDay, daysOfWeek(r.Pattern.DaysOfWeek))
	case "RELATIVEYEARLY":
		// Event repeats on the specified day or days of the week, in the same relative position in a specific month of the year,
		// based on the number of years between occurrences.
		rule.Add(RuleFrequency, "YEARLY")
		rule.Add(RuleByMonth, strconv.Itoa(r.Pattern.Month))
	case "ABSOLUTEYEARLY":
		// Event repeats on the specified day and month, based on the number of years between occurrences.
		rule.Add(RuleFrequency, "YEARLY")
		rule.Add(RuleByMonth, strconv.Itoa(r.Pattern.Month))
		rule.Add(RuleByMonthDay, strconv.Itoa(r.Pattern.DayOfMonth))
	case "RELATIVEMONTHLY":
		// Event repeats on the specified day or days of the week, in the same relative position in the month,
		// based on the number of months between occurrences.
		rule.Add(RuleFrequency, "MONTHLY")
	case "ABSOLUTEMONTHLY":
		// Event repeats on the specified day of the month (e.g. the 15th), based on the number of months between occurrences.
		rule.Add(RuleFrequency, "MONTHLY")
		rule.Add(RuleByMonthDay, weekIndex[r.Pattern.Index]+strconv.Itoa(r.Pattern.DayOfMonth))
	default:
		panic("unsupported value " + r.Pattern.Type)
	}

	rule.Add(RuleInterval, strconv.Itoa(r.Pattern.Interval))

	if len(r.Pattern.FirstDayOfWeek) >= 2 {
		rule.Add(RuleWeekStart, r.Pattern.FirstDayOfWeek[:2])
	}

	switch r.Range.Type {
	case azure.RecurrenceEndDate:
		rule.Add(RuleUntil, utctime(r.Range.EndDate.Time()))
	case azure.RecurrenceNumbered:
		rule.Add(RuleCount, strconv.Itoa(r.Range.NumberOfOccurrences))
	case azure.RecurrenceNoEnd:
	default:
	}

	rrule := rule.Serialize()

	e.AddRrule(rrule)
}
