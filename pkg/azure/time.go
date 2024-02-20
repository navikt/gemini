package azure

import (
	"time"
)

// https://docs.microsoft.com/en-us/graph/api/resources/patternedrecurrence?view=graph-rest-1.0
type PatternedRecurrence struct {
	Pattern RecurrencePattern
	Range   RecurrenceRange
}

// For access reviews:
// Do not specify this property for a one-time access review.
// Only interval, dayOfMonth, and type (weekly, absoluteMonthly) properties of recurrencePattern are supported.
// https://docs.microsoft.com/en-us/graph/api/resources/recurrencepattern?view=graph-rest-1.0
type RecurrencePattern struct {
	DayOfMonth     int      `json:"dayOfMonth"`
	DaysOfWeek     []string `json:"daysOfWeek"`
	FirstDayOfWeek string   `json:"firstDayOfWeek"`
	Index          string   `json:"index"`
	Interval       int      `json:"interval"`
	Month          int      `json:"month"`
	Type           string   `json:"type"`
}

const (
	RecurrenceEndDate  = "endDate"
	RecurrenceNoEnd    = "noEnd"
	RecurrenceNumbered = "numbered"
)

// https://docs.microsoft.com/en-us/graph/api/resources/recurrencerange?view=graph-rest-1.0
type RecurrenceRange struct {
	EndDate             SimpleDate `json:"endDate"`
	NumberOfOccurrences int        `json:"numberOfOccurrences"`
	RecurrenceTimeZone  string     `json:"recurrenceTimeZone"`
	StartDate           SimpleDate `json:"startDate"`
	Type                string     `json:"type"` // The recurrence range. The possible values are: endDate, noEnd, numbered. Required.
}

type SimpleDate string

const simpleDate = "2006-01-02"

func (m SimpleDate) Time() time.Time {
	t, _ := time.Parse(simpleDate, string(m))
	return t
}

// Microsoft's idea of a datetime object:
// 1. Start out with RFC3339, which includes a timezone.
// 2. Remove timezone, because who needs standards anyway.
// 3. Reintroduce timezone in a separate variable and bundle the two together in a struct.
//
// However, this distinction is probably needed in this case, because time zone rules can change.
// Especially future timestamps (as used by events) are afflicted by this fact.
type TimeAndZone struct {
	DateTime string
	TimeZone string
}

// RFC3339 without timezone.
const rfc3339WithoutTimezoneFormat = "2006-01-02T15:04:05.0000000"

// Parse timestamp according to a timezone.
//
//	"start": {
//	  "dateTime": "2022-01-11T11:00:00.0000000",
//	  "timeZone": "UTC"
//	},
func (m *TimeAndZone) Time() time.Time {
	var loc *time.Location
	loc, err := time.LoadLocation(m.TimeZone)
	if err != nil {
		loc = time.UTC
	}
	t, err := time.ParseInLocation(rfc3339WithoutTimezoneFormat, m.DateTime, loc)
	if err != nil {
		panic(err)
	}
	return t
}
