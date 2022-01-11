package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/ambientsound/gemini/pkg/azure"
	"github.com/ambientsound/gemini/pkg/calserv"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

func main() {
	err := run()
	if err != nil {
		log.Errorf("fatal: %s\n", err)
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

func getallevents(client *http.Client) ([]azure.Event, error) {
	var events []azure.Event

	log.Info("Loading events...")
	t := time.Now()
	defer log.Infof("Got %d events in %s", len(events), time.Since(t))

	uri := "https://graph.microsoft.com/v1.0/me/calendar/events?$top=100"
	resultset := make([]azure.Event, 0, 8192)

	for len(uri) > 0 {
		log.Infof("Fetching events from %s", uri)

		resp, err := client.Get(uri)
		if err != nil {
			return nil, err
		}

		//goland:noinspection ALL
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("API returned %s: %s", resp.Status, body)
		}

		payload := &azure.Result{}
		err = json.NewDecoder(resp.Body).Decode(payload)
		if err != nil {
			return nil, err
		}

		resultset = append(resultset, payload.Value...)

		uri = payload.Next
	}

	return resultset, nil
}

func synccalendar(ctx context.Context) ([]azure.Event, error) {
	var token *oauth2.Token
	var err error
	var events []azure.Event

	evdata, err := ioutil.ReadFile("/tmp/gemini.events")
	if err == nil {
		events = make([]azure.Event, 0)
		err = json.Unmarshal(evdata, &events)
		if err == nil {
			return events, nil
		}
	}

	tok, err := ioutil.ReadFile("/tmp/gemini.token")

	if err == nil {
		token = &oauth2.Token{}
		err = json.Unmarshal(tok, token)
	}

	if err != nil {
		token, err = authtoken(ctx)
		if err != nil {
			return nil, fmt.Errorf("auth: %w", err)
		}
		data, _ := json.Marshal(token)
		ioutil.WriteFile("/tmp/gemini.token", data, 0600)
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	log.Info("Microsoft Graph API client instantiated")

	events, err = getallevents(client)

	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(events)
	ioutil.WriteFile("/tmp/gemini.events", data, 0600)

	return events, err
}

func run() error {
	log.SetOutput(os.Stderr)
	log.Infof("GEMINI starting up - Office365 to iCal")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Infof("Synchronizing calendar from Azure...")
	events, err := synccalendar(ctx)
	if err != nil {
		return err
	}

	log.Infof("Loaded %d events from Azure", len(events))
	cal := calserv.Calendar(events)

	srv := &calserv.Server{}
	srv.SetCalendar(cal)

	log.Infof("Web server started, ready to receive requests.")
	return http.ListenAndServe("127.0.0.1:9999", srv)
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
