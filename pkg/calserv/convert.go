package calserv

import (
	ics "github.com/arran4/golang-ical"
	"github.com/nais/gemini/pkg/azure"
)

const almostRFCTime = "20060102T150405"
const tz = "/Europe/Oslo"

func Calendar(events []azure.Event) *ics.Calendar {
	cal := ics.NewCalendar()
	for _, ev := range events {
		cal.AddVEvent(Convert(ev))
	}
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
	e.SetDescription(event.BodyPreview)
	e.SetLocation(event.Location.DisplayName)
	e.SetOrganizer(event.Organizer.EmailAddress.Name, &ics.KeyValues{
		Key:   "MAILTO",
		Value: []string{event.Organizer.EmailAddress.Address},
	})

	// optimalization for Google Calendar limit on 1MB per iCAL file
	const maxAttendees = 10
	if len(event.Attendees) <= maxAttendees {
		for _, att := range event.Attendees {
			e.AddAttendee(att.EmailAddress.Address)
		}
	}

	// TODO
	//e.AddAlarm()

	e.SetURL(event.WebLink)
	setRecurrence(event, e)

	return e
}
