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

type BullshitTime struct {
	DateTime string
	TimeZone string
}

const bullshitTime = "2006-01-02T15:04:05.0000000"

// Time converts bullshit data into real timestamps
// "start": {
//   "dateTime": "2022-01-11T11:00:00.0000000",
//   "timeZone": "UTC"
// },
func (m *BullshitTime) Time() time.Time {
	var loc *time.Location
	loc, err := time.LoadLocation(m.TimeZone)
	if err != nil {
		loc = time.UTC
	}
	t, err := time.ParseInLocation(bullshitTime, m.DateTime, loc)
	if err != nil {
		panic(err)
	}
	//t, _ := time.Parse(bullshitTime, m.DateTime)
	//return t.In(loc)
	return t
}
