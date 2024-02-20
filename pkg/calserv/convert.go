package calserv

import (
	"strings"

	ics "github.com/arran4/golang-ical"

	"github.com/nais/gemini/pkg/azure"
)

const almostRFCTime = "20060102T150405"
const tz = "/Europe/Oslo"
const geminiEmoji = "♊"

func Calendar(events []azure.Event) *ics.Calendar {
	cal := ics.NewCalendar()
	for _, ev := range events {
		cal.AddVEvent(Convert(ev))
	}
	cal.SetName("NAV")
	cal.SetProductId("GEMINI")
	cal.SetTzid(tz)
	return cal
}

func Convert(event azure.Event) *ics.VEvent {
	e := ics.NewEvent(event.Id)
	tz := &ics.KeyValues{
		Key:   "TZID",
		Value: []string{tz},
	}

	e.SetSummary(event.Subject)

	if event.IsAllDay {
		e.SetAllDayStartAt(event.Start.Time(), tz)
		e.SetAllDayEndAt(event.End.Time(), tz)
	} else {
		e.SetProperty(ics.ComponentPropertyDtStart, event.Start.Time().Local().Format(almostRFCTime), tz)
		e.SetProperty(ics.ComponentPropertyDtEnd, event.End.Time().Local().Format(almostRFCTime), tz)
	}

	description := ""
	if len(event.Body.Content) > 0 && event.Body.ContentType == "text" {
		description = fixText(event.Body.Content)
	} else if len(event.BodyPreview) > 0 {
		description = fixText(event.BodyPreview)
	}

	e.SetLocation(event.Location.DisplayName)
	e.SetOrganizer(event.Organizer.EmailAddress.Name, &ics.KeyValues{
		Key:   "MAILTO",
		Value: []string{event.Organizer.EmailAddress.Address},
	})

	for _, att := range event.Attendees {
		e.AddAttendee(att.EmailAddress.Address)
	}

	if event.ResponseStatus.Response == azure.ResponseStatusNotResponded {
		description = geminiEmoji + " RSVP!\n\n" + description
	}

	e.SetDescription(description)

	// TODO
	//e.AddAlarm()

	e.SetURL(event.WebLink)
	setRecurrence(event, e)

	return e
}

// Remove crappy Microsoft formatting
func fixText(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)
	return s
}
