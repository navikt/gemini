package calserv

import (
	"fmt"
	"strings"
	"time"

	"github.com/ambientsound/gemini/pkg/azure"
	ics "github.com/arran4/golang-ical"
)

func Calendar(events []azure.Event) *ics.Calendar {
	cal := ics.NewCalendar()
	for _, ev := range events {
		cal.AddVEvent(Convert(ev))
	}
	return cal
}

func ConvertMany(events []azure.Event) []*ics.VEvent {
	icsevents := make([]*ics.VEvent, len(events))
	for i, ev := range events {
		icsevents[i] = Convert(ev)
	}
	return icsevents
}

func utctime(t time.Time) string {
	const format = "20060102T150405Z"
	return t.Format(format)
}

func setRecurrence(event azure.Event, e *ics.VEvent) {
	if event.Recurrence == nil {
		return
	}
	r := event.Recurrence

	rules := make(RRules)
	rules.Add("FREQ", r.Pattern.Type)
	rules.Add("INTERVAL", r.Pattern.Interval)
	if len(r.Pattern.FirstDayOfWeek) >= 2 {
		rules.Add("WKST", r.Pattern.FirstDayOfWeek[:2])
	}

	switch r.Range.Type {
	case azure.RecurrenceEndDate:
		rules.Add("UNTIL", utctime(r.Range.EndDate.Time()))
	case azure.RecurrenceNumbered:
		rules.Add("COUNT", r.Range.NumberOfOccurrences)
	case azure.RecurrenceNoEnd:
	default:
	}

	rrule := rules.Serialize()
	e.SetDescription(rrule)

	e.AddRrule(rrule)
}

type RRules map[string]interface{}

func (r RRules) Serialize() string {
	parts := make([]string, 0, len(r))
	for k, v := range r {
		parts = append(parts, strings.ToUpper(fmt.Sprintf("%v=%v", k, v)))
	}
	return strings.Join(parts, ";")
}

func (r RRules) Add(k string, v interface{}) {
	r[k] = v
}

func Convert(event azure.Event) *ics.VEvent {
	e := ics.NewEvent(event.Id)
	e.SetSummary(event.Subject)
	if event.IsAllDay {
		e.SetAllDayStartAt(event.Start.Time())
		e.SetAllDayEndAt(event.End.Time())
	} else {
		e.SetStartAt(event.Start.Time())
		e.SetEndAt(event.End.Time())
	}
	e.SetDescription(event.BodyPreview)
	e.SetLocation(event.Location.DisplayName)
	//e.SetOrganizer()
	//e.AddAlarm()
	e.SetURL(event.WebLink)
	setRecurrence(event, e)
	return e
}
