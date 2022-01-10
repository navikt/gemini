package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

type odata interface{}

type Result struct {
	Value []Event
}

type MicrosoftTime struct {
	DateTime string
	TimeZone string
}

type Event struct {
	AllowNewTimeProposals         bool
	Attendees                     []odata       //": [{"@odata.type": "microsoft.graph.attendee"}],
	Body                          odata         //": {"@odata.type": "microsoft.graph.itemBody"},
	BodyPreview                   string        //": "string",
	Categories                    []string      //": ["string"],
	ChangeKey                     string        //": "string",
	CreatedDateTime               string        //": "String (timestamp)",
	End                           MicrosoftTime //": {"@odata.type": "microsoft.graph.dateTimeTimeZone"},
	HasAttachments                bool          //": true,
	HideAttendees                 bool          //": false,
	Id                            string        //": "string (identifier)",
	Importance                    string        //": "String",
	IsAllDay                      bool          //": true,
	IsCancelled                   bool          //": true,
	IsDraft                       bool          //": false,
	IsOnlineMeeting               bool          //": true,
	IsOrganizer                   bool          //": true,
	IsReminderOn                  bool          //": true,
	LastModifiedDateTime          string        //": "String (timestamp)",
	Location                      odata         //": {"@odata.type": "microsoft.graph.location"},
	Locations                     []odata       //": [{"@odata.type": "microsoft.graph.location"}],
	OnlineMeeting                 odata         //": {"@odata.type": "microsoft.graph.onlineMeetingInfo"},
	OnlineMeetingProvider         string        //": "string",
	OnlineMeetingUrl              string        //": "string",
	Organizer                     odata         //": {"@odata.type": "microsoft.graph.recipient"},
	OriginalEndTimeZone           string        //": "string",
	OriginalStart                 string        //": "String (timestamp)",
	OriginalStartTimeZone         string        //": "string",
	Recurrence                    odata         //": {"@odata.type": "microsoft.graph.patternedRecurrence"},
	ReminderMinutesBeforeStart    int           //": 1024,
	ResponseRequested             bool          //": true,
	ResponseStatus                odata         //": {"@odata.type": "microsoft.graph.responseStatus"},
	Sensitivity                   string        //": "String",
	SeriesMasterId                string        //": "string",
	ShowAs                        string        //": "String",
	Start                         MicrosoftTime //": {"@odata.type": "microsoft.graph.dateTimeTimeZone"},
	Subject                       string        //": "string",
	Type                          string        //": "String",
	WebLink                       string        //": "string",
	Attachments                   []odata       //[ { "@odata.type": "microsoft.graph.attachment" } ],
	Calendar                      odata         //": { "@odata.type": "microsoft.graph.calendar" },
	Extensions                    []odata       //": [ { "@odata.type": "microsoft.graph.extension" } ],
	Instances                     []odata       //": [ { "@odata.type": "microsoft.graph.event" }],
	SingleValueExtendedProperties []odata       //": [ { "@odata.type": "microsoft.graph.singleValueLegacyExtendedProperty" }],
	MultiValueExtendedProperties  []odata       //": [ { "@odata.type": "microsoft.graph.multiValueLegacyExtendedProperty" }]
}

// start, end (string datetime)
// type, recurrence
// subject
// location.displayName

func main() {
	err := run()
	if err != nil {
		fmt.Printf("fatal: %s\n", err)
		os.Exit(1)
	}
}

type callbackHandler struct {
	c chan authresult
}

type authresult struct {
	code  string
	state string
	err   error
}

func (c *callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	q := r.URL.Query()
	errid := q.Get("error")
	errstr := q.Get("error_description")
	if len(errid) > 0 {
		err = fmt.Errorf("%s: %s", errid, errstr)
	}
	c.c <- authresult{
		code:  q.Get("code"),
		state: q.Get("state"),
		err:   err,
	}
}

func run() error {
	var token *oauth2.Token
	var err error

	log.SetOutput(os.Stderr)
	ctx := context.Background()

	tok, err := ioutil.ReadFile("/tmp/gemini.token")

	if err == nil {
		token = &oauth2.Token{}
		err = json.Unmarshal(tok, token)
	}

	if err != nil {
		token, err = authtoken(ctx)
		if err != nil {
			return fmt.Errorf("auth: %w", err)
		}
		data, _ := json.Marshal(token)
		ioutil.WriteFile("/tmp/gemini.token", data, 0600)
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	log.Info("Microsoft Graph API client instantiated")

	resp, err := client.Get("https://graph.microsoft.com/v1.0/me/calendar/events?$top=100")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//io.Copy(os.Stdout, resp.Body)
	//return nil

	payload := &Result{}
	err = json.NewDecoder(resp.Body).Decode(payload)
	if err != nil {
		return err
	}

	for _, ev := range payload.Value {
		log.Infof("%s: %s", ev.Start.DateTime, ev.Subject)
	}

	return nil
}

func authtoken(ctx context.Context) (*oauth2.Token, error) {
	redirectURL := "http://localhost:12345"
	scopes := []string{"openid", "Calendars.Read"}

	cfg := &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		Endpoint:     endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}

	handler := func(authCodeURL string) (code string, state string, err error) {
		log.Info(authCodeURL)
		c := make(chan authresult)
		server := &http.Server{
			Addr:    "127.0.0.1:12345",
			Handler: &callbackHandler{c: c},
		}
		defer server.Close()
		defer close(c)
		go server.ListenAndServe()
		reply := <-c
		return reply.code, reply.state, reply.err
	}

	code, _, err := handler(cfg.AuthCodeURL("mystate"))
	if err != nil {
		return nil, err
	}

	return cfg.Exchange(ctx, code)
}
