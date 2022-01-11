package calserv

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ambientsound/gemini/pkg/azure"
	ics "github.com/arran4/golang-ical"
	log "github.com/sirupsen/logrus"
)

func Calendar(events []azure.Event) *ics.Calendar {
	cal := ics.NewCalendar()
	for _, ev := range events {
		cal.AddVEvent(Convert(ev))
	}
	cal.SetProductId("GEMINI")
	cal.SetTzid("/Europe/Oslo")
	return cal
}

func utctime(t time.Time) string {
	const format = "20060102T150405Z"
	return t.Local().Format(format)
}

func setRecurrence(event azure.Event, e *ics.VEvent) {
	if event.Recurrence == nil {
		return
	}
	r := event.Recurrence

	rules := &RRules{
		rules: []string{},
	}

	switch strings.ToUpper(r.Pattern.Type) {
	case "SECONDLY":
	case "MINUTELY":
	case "HOURLY":
	case "DAILY":
	case "WEEKLY":
	case "MONTHLY":
	case "YEARLY":
	case "RELATIVEMONTHLY": // Microsoft bullshit again
		log.Warnf("Converting %s to MONTHLY", r.Pattern.Type)
		r.Pattern.Type = "MONTHLY"
	default:
		panic("unsupported value " + r.Pattern.Type)
	}
	//     freq       = "SECONDLY" / "MINUTELY" / "HOURLY" / "DAILY"
	//                / "WEEKLY" / "MONTHLY" / "YEARLY"
	rules.Add("FREQ", r.Pattern.Type)
	rules.Add("INTERVAL", strconv.Itoa(r.Pattern.Interval))

	if len(r.Pattern.FirstDayOfWeek) >= 2 {
		rules.Add("WKST", r.Pattern.FirstDayOfWeek[:2])
	}

	switch r.Range.Type {
	case azure.RecurrenceEndDate:
		rules.Add("UNTIL", utctime(r.Range.EndDate.Time()))
	case azure.RecurrenceNumbered:
		rules.Add("COUNT", strconv.Itoa(r.Range.NumberOfOccurrences))
	case azure.RecurrenceNoEnd:
	default:
	}

	rrule := rules.Serialize()
	//e.SetDescription(rrule)

	e.AddRrule(rrule)
}

type RRules struct {
	rules []string
}

func (r RRules) Serialize() string {
	return strings.Join(r.rules, ";")
}

func (r *RRules) Add(k, v string) {
	r.rules = append(r.rules, strings.ToUpper(fmt.Sprintf("%v=%v", k, v)))
}

func Convert(event azure.Event) *ics.VEvent {
	e := ics.NewEvent(event.Id)
	tz := &ics.KeyValues{
		Key: "TZID",
		Value: []string{
			"/Europe/Oslo",
		},
	}

	e.SetSummary(event.Subject)

	const lureformat = "20060102T150405"
	if event.IsAllDay {
		e.SetAllDayStartAt(event.Start.Time(), tz)
		e.SetAllDayEndAt(event.End.Time(), tz)
	} else {
		e.SetProperty(ics.ComponentPropertyDtStart, event.Start.Time().Local().Format(lureformat), tz)
		e.SetProperty(ics.ComponentPropertyDtEnd, event.End.Time().Local().Format(lureformat), tz)
		//e.SetStartAt(event.Start.Time(), tz)
		//e.SetEndAt(event.End.Time(), tz)
	}
	e.SetDescription(event.BodyPreview)
	e.SetLocation(event.Location.DisplayName)
	//e.SetOrganizer()
	//e.AddAlarm()
	e.SetURL(event.WebLink)
	setRecurrence(event, e)

	return e
}
