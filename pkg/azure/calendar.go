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

// https://docs.microsoft.com/en-us/graph/api/resources/responsestatus?view=graph-rest-1.0
type ResponseStatus struct {
	// The response type. Possible values are: none, organizer, tentativelyAccepted, accepted, declined, notResponded.
	Response string
}

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
	Body                          odata                //": {"@odata.type": "microsoft.graph.itemBody"},
	BodyPreview                   string               //": "string",
	Categories                    []string             //": ["string"],
	ChangeKey                     string               //": "string",
	CreatedDateTime               string               //": "String (timestamp)",
	End                           BullshitTime         //": {"@odata.type": "microsoft.graph.dateTimeTimeZone"},
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
	ResponseStatus                odata                //": {"@odata.type": "microsoft.graph.responseStatus"},
	Sensitivity                   string               //": "String",
	SeriesMasterId                string               //": "string",
	ShowAs                        string               //": "String",
	Start                         BullshitTime         //": {"@odata.type": "microsoft.graph.dateTimeTimeZone"},
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

// List calendar events at Azure endpoint.
//
// The events are filtered such that they either have to be recurring events,
// or at most six months before today.
//
// https://learn.microsoft.com/en-us/graph/api/group-list-events?view=graph-rest-1.0&tabs=http
func GetCalendarEvents(client *http.Client) ([]Event, error) {
	t := time.Now()

	const backfillDuration = time.Hour * 24 * 30 * 6
	cutoffTime := t.Add(-backfillDuration)

	values := &url.Values{}
	values.Set("$top", "100")
	values.Set("$filter", "(start/dateTime ge '"+cutoffTime.Format(time.RFC3339)+"') or (type eq 'seriesMaster')")
	uri := "https://graph.microsoft.com/v1.0/me/calendar/events?" + values.Encode()

	resultset := make([]Event, 0, 32)

	for len(uri) > 0 {
		log.Debugf("Fetching events from %s", uri)

		// TODO: add request header for longer description, but as text
		// Prefer: outlook.body-content-type="text"

		resp, err := client.Get(uri)
		if err != nil {
			return nil, err
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
