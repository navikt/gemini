package calserv

import (
	"fmt"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"

	"github.com/nais/gemini/pkg/azure"
	"github.com/nais/gemini/pkg/version"
)

const almostRFCTime = "20060102T150405"
const tz = "/Europe/Oslo"
const geminiEmoji = "♊"

func Calendar(events []azure.Event, converter func(azure.Event) *ics.VEvent) *ics.Calendar {
	cal := ics.NewCalendar()

	cal.SetName("Nav")
	cal.SetTzid(tz)

	// Product identifier is supposed to say whose software made this file.
	// Documentation: https://icalendar.org/iCalendar-RFC-5545/3-7-3-product-identifier.html
	// "The following is an example of this property. It does not imply that English is the default language."
	// PRODID:-//ABC Corporation//NONSGML My Product//EN
	cal.SetProductId(fmt.Sprintf("-//nais.io//NONSGML Gemini %s//EN", version.Version()))

	for _, ev := range events {
		cal.AddVEvent(converter(ev))
	}

	cal.SetLastModified(time.Now())

	return cal
}

// Convert an Azure event to an ICS event, but strip all information
// and replace it with a generic "busy".
func ConvertPublic(event azure.Event) *ics.VEvent {
	e := ics.NewEvent(event.Id)
	tz := &ics.KeyValues{
		Key:   "TZID",
		Value: []string{tz},
	}

	e.SetSummary("Opptatt")

	if event.IsAllDay {
		e.SetAllDayStartAt(event.Start.Time(), tz)
		e.SetAllDayEndAt(event.End.Time(), tz)
	} else {
		e.SetProperty(ics.ComponentPropertyDtStart, event.Start.Time().Local().Format(almostRFCTime), tz)
		e.SetProperty(ics.ComponentPropertyDtEnd, event.End.Time().Local().Format(almostRFCTime), tz)
	}

	// TODO: do we want tentative or unanswered events to pop up in the public calendar?
	if event.ResponseStatus.Response == azure.ResponseStatusNotResponded {
	}

	setRecurrence(event, e)

	return e
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
