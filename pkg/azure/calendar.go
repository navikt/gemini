package azure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

type odata interface{}

type Location struct {
	DisplayName string
}

// https://docs.microsoft.com/en-us/graph/api/resources/recipient?view=graph-rest-1.0
type Recipient struct {
	EmailAddress EmailAddress
}

// https://docs.microsoft.com/en-us/graph/api/resources/emailaddress?view=graph-rest-1.0
type EmailAddress struct {
	Address string
	Name    string
}

// https://docs.microsoft.com/en-us/graph/api/resources/attendee?view=graph-rest-1.0
type Attendee struct {
	EmailAddress EmailAddress
	Status       ResponseStatus
	Type         string
}

// https://learn.microsoft.com/en-us/graph/api/resources/itembody?view=graph-rest-1.0
type Body struct {
	Content     string // The content of the item.
	ContentType string // The type of the content. Possible values are text and html.
}

// https://docs.microsoft.com/en-us/graph/api/resources/responsestatus?view=graph-rest-1.0
type ResponseStatus struct {
	// The response type. Possible values are: none, organizer, tentativelyAccepted, accepted, declined, notResponded.
	Response string
}

const ResponseStatusNone = "none"
const ResponseStatusOrganizer = "organizer"
const ResponseStatusTentativelyAccepted = "tentativelyAccepted"
const ResponseStatusAccepted = "accepted"
const ResponseStatusNotResponded = "notResponded"
const ResponseStatusDeclined = "declined"

type Result struct {
	Next  string `json:"@odata.nextLink"`
	Value Events
}

type Events []Event

// https://docs.microsoft.com/en-us/graph/api/calendar-list-events?view=graph-rest-1.0&tabs=http
// https://docs.microsoft.com/en-us/graph/api/resources/event?view=graph-rest-1.0
type Event struct {
	AllowNewTimeProposals         bool
	Attendees                     []Attendee           //": [{"@odata.type": "microsoft.graph.attendee"}],
	Body                          Body                 //": {"@odata.type": "microsoft.graph.itemBody"},
	BodyPreview                   string               //": "string",
	Categories                    []string             //": ["string"],
	ChangeKey                     string               //": "string",
	CreatedDateTime               string               //": "String (timestamp)",
	End                           TimeAndZone          //": {"@odata.type": "microsoft.graph.dateTimeTimeZone"},
	HasAttachments                bool                 //": true,
	HideAttendees                 bool                 //": false,
	Id                            string               //": "string (identifier)",
	Importance                    string               //": "String",
	IsAllDay                      bool                 //": true,
	IsCancelled                   bool                 //": true,
	IsDraft                       bool                 //": false,
	IsOnlineMeeting               bool                 //": true,
	IsOrganizer                   bool                 //": true,
	IsReminderOn                  bool                 //": true,
	LastModifiedDateTime          string               //": "String (timestamp)",
	Location                      Location             //": {"@odata.type": "microsoft.graph.location"},
	Locations                     []odata              //": [{"@odata.type": "microsoft.graph.location"}],
	OnlineMeeting                 odata                //": {"@odata.type": "microsoft.graph.onlineMeetingInfo"},
	OnlineMeetingProvider         string               //": "string",
	OnlineMeetingUrl              string               //": "string",
	Organizer                     Recipient            //": {"@odata.type": "microsoft.graph.recipient"},
	OriginalEndTimeZone           string               //": "string",
	OriginalStart                 string               //": "String (timestamp)",
	OriginalStartTimeZone         string               //": "string",
	Recurrence                    *PatternedRecurrence //": {"@odata.type": "microsoft.graph.patternedRecurrence"},
	ReminderMinutesBeforeStart    int                  //": 1024,
	ResponseRequested             bool                 //": true,
	ResponseStatus                ResponseStatus       //": {"@odata.type": "microsoft.graph.responseStatus"},
	Sensitivity                   string               //": "String",
	SeriesMasterId                string               //": "string",
	ShowAs                        string               //": "String",
	Start                         TimeAndZone          //": {"@odata.type": "microsoft.graph.dateTimeTimeZone"},
	Subject                       string               //": "string",
	Type                          string               //": "String",
	WebLink                       string               //": "string",
	Attachments                   []odata              //[ { "@odata.type": "microsoft.graph.attachment" } ],
	Calendar                      odata                //": { "@odata.type": "microsoft.graph.calendar" },
	Extensions                    []odata              //": [ { "@odata.type": "microsoft.graph.extension" } ],
	Instances                     []odata              //": [ { "@odata.type": "microsoft.graph.event" }],
	SingleValueExtendedProperties []odata              //": [ { "@odata.type": "microsoft.graph.singleValueLegacyExtendedProperty" }],
	MultiValueExtendedProperties  []odata              //": [ { "@odata.type": "microsoft.graph.multiValueLegacyExtendedProperty" }]
}

// How long into the past and future do we peek?
const backfillDuration = time.Hour * 24 * 30 * 2 // two months
const forwardfillDuration = time.Hour * 24 * 365 // one year

// List calendar events at Azure endpoint.
//
// The events are filtered such that they either have to be recurring events,
// or at most two months before today.
//
// https://learn.microsoft.com/en-us/graph/api/group-list-events?view=graph-rest-1.0&tabs=http
func listCalendarEvents(client *http.Client) ([]Event, error) {
	t := time.Now()

	cutoffTime := t.Add(-backfillDuration)

	values := &url.Values{}
	values.Set("$top", "100")
	values.Set("$filter", "(end/dateTime ge '"+cutoffTime.Format(time.RFC3339)+"') or (type eq 'seriesMaster')")
	uri := "https://graph.microsoft.com/v1.0/me/calendar/events?" + values.Encode()

	resultset := make([]Event, 0, 32)

	for len(uri) > 0 {
		log.Tracef("Fetching events from %s", uri)

		req, err := http.NewRequest(http.MethodGet, uri, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Prefer", "outlook.body-content-type=text")

		resp, err := client.Do(req)
		if err != nil {
			return nil, DecodeOauth2ApiError(err)
		}

		//goland:noinspection ALL
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API returned %s: %s", resp.Status, body)
		}

		payload := &Result{}
		err = json.NewDecoder(resp.Body).Decode(payload)
		if err != nil {
			return nil, err
		}

		resultset = append(resultset, payload.Value...)

		uri = payload.Next
	}

	defer log.Debugf("Fetched %d events in %s", len(resultset), time.Since(t))

	return resultset, nil
}

func GetCalendarEvents(client *http.Client) ([]Event, error) {
	events, err := listCalendarEvents(client)
	if err != nil {
		return events, err
	}

	results := make([]Event, 0, len(events))

	// After the events have been fetched, they must be enriched,
	// in order to know if instances of a recurring event has been cancelled.
	for _, event := range events {
		if event.Type != "seriesMaster" {
			results = append(results, event)
			continue
		}

		instances, err := GetRecurringEventInstances(client, event.Id)
		if err != nil {
			return nil, err
		}

		if len(instances) == 0 {
			continue
		}

		log.Debugf("%2d instances of %s", len(instances), event.Id)
		results = append(results, instances...)
	}

	return results, err
}
