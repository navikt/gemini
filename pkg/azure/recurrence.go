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

// Instances of recurring events can be changed or deleted, but this info is not included when listing events.
// Thus, we have to run a query _per recurring event_ to accurately know which events to render.
// However, because of Microsoft's poor API design they will just have to deal with that.
//
// https://learn.microsoft.com/en-us/graph/api/event-list-instances?view=graph-rest-1.0&tabs=http
func GetRecurringEventInstances(client *http.Client, eventID string) ([]Event, error) {
	t := time.Now()
	fromTime := t.Add(-backfillDuration)
	untilTime := t.Add(forwardfillDuration)

	values := &url.Values{}
	values.Set("$top", "100")
	//values.Set("$filter", "type eq 'exception'")
	values.Set("startDateTime", fromTime.Format(time.RFC3339))
	values.Set("endDateTime", untilTime.Format(time.RFC3339))
	uri := "https://graph.microsoft.com/v1.0/me/calendar/events/" + eventID + "/instances?" + values.Encode()

	resultset := make([]Event, 0, 32)

	for len(uri) > 0 {
		log.Debugf("Fetching event instances from %s", uri)

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

	defer log.Debugf("Fetched %d event instances in %s", len(resultset), time.Since(t))

	return resultset, nil
}
