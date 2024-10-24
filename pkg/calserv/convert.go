package calserv

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"

	"github.com/nais/gemini/pkg/azure"
	"github.com/nais/gemini/pkg/version"
)

const almostRFCTime = "20060102T150405"
const tz = "/Europe/Oslo"
const geminiEmoji = "♊"

func AzureCalendar(events []azure.Event, converter func(azure.Event) *ics.VEvent) *ics.Calendar {
	icsEvents := make([]*ics.VEvent, 0, len(events))
	for _, ev := range events {
		icsEvents = append(icsEvents, converter(ev))
	}
	return Calendar(icsEvents)
}

func Calendar(events []*ics.VEvent) *ics.Calendar {
	cal := ics.NewCalendar()

	cal.SetName("Nav")
	cal.SetTzid(tz)

	// Product identifier is supposed to say whose software made this file.
	// Documentation: https://icalendar.org/iCalendar-RFC-5545/3-7-3-product-identifier.html
	// "The following is an example of this property. It does not imply that English is the default language."
	// PRODID:-//ABC Corporation//NONSGML My Product//EN
	cal.SetProductId(fmt.Sprintf("-//nais.io//NONSGML Gemini %s//EN", version.Version()))

	for _, ev := range events {
		cal.AddVEvent(ev)
	}

	cal.SetLastModified(time.Now())

	return cal
}

// Convert an Azure event to an ICS event, but strip all information
// and replace it with a generic "busy".
func ConvertPublic(event azure.Event) *ics.VEvent {
	e := ics.NewEvent(event.Id)
	timezone := &ics.KeyValues{
		Key:   "TZID",
		Value: []string{tz},
	}

	e.SetSummary("Opptatt")

	if event.IsAllDay {
		e.SetAllDayStartAt(event.Start.Time(), timezone)
		e.SetAllDayEndAt(event.End.Time(), timezone)
	} else {
		e.SetProperty(ics.ComponentPropertyDtStart, event.Start.Time().Local().Format(almostRFCTime), timezone)
		e.SetProperty(ics.ComponentPropertyDtEnd, event.End.Time().Local().Format(almostRFCTime), timezone)
	}

	// TODO: do we want tentative or unanswered events to pop up in the public calendar?
	if event.ResponseStatus.Response == azure.ResponseStatusNotResponded {
	}

	setRecurrence(event, e)

	return e
}

// Generate a calendar that shows the associated error message
// every day during core working hours (0900-1430).
func ErrorCalendarWeek(msg string) *ics.Calendar {
	// Find midnight of today
	cursor := time.Now().Local()
	cursor = time.Date(
		cursor.Year(),
		cursor.Month(),
		cursor.Day(),
		0, 0, 0, 0,
		time.Now().Location(),
	)

	// Travel back in time to find this week's Monday
	for cursor.Weekday() != time.Monday {
		cursor = cursor.Add(-24 * time.Hour)
	}

	// By now we should have reached Monday.
	icsEvents := make([]*ics.VEvent, 0, 5)
	i := 5
	for i > 0 {
		icsEvents = append(icsEvents, ErrorMessage(msg, cursor))
		cursor = cursor.Add(24 * time.Hour)
		i--
	}

	return Calendar(icsEvents)
}

func ErrorMessage(msg string, date time.Time) *ics.VEvent {
	e := ics.NewEvent(strconv.Itoa(rand.Int()))
	timezone := &ics.KeyValues{
		Key:   "TZID",
		Value: []string{tz},
	}

	date = date.Local()
	start := date.Truncate(time.Hour).Add(9 * time.Hour)
	end := date.Truncate(time.Hour).Add(14 * time.Hour).Add(30 * time.Minute)

	e.SetProperty(ics.ComponentPropertyDtStart, start.Format(almostRFCTime), timezone)
	e.SetProperty(ics.ComponentPropertyDtEnd, end.Format(almostRFCTime), timezone)

	e.SetOrganizer("Gemini")
	e.SetSummary(msg)
	e.SetDescription(fmt.Sprintf("Gemini cannot sync your calendar due to the following error:\n\n%s", msg))

	return e
}

func Convert(event azure.Event) *ics.VEvent {
	e := ics.NewEvent(event.Id)
	timezone := &ics.KeyValues{
		Key:   "TZID",
		Value: []string{tz},
	}

	e.SetSummary(event.Subject)

	if event.IsAllDay {
		e.SetAllDayStartAt(event.Start.Time(), timezone)
		e.SetAllDayEndAt(event.End.Time(), timezone)
	} else {
		e.SetProperty(ics.ComponentPropertyDtStart, event.Start.Time().Local().Format(almostRFCTime), timezone)
		e.SetProperty(ics.ComponentPropertyDtEnd, event.End.Time().Local().Format(almostRFCTime), timezone)
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
