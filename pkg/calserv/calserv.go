package calserv

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/nais/gemini/pkg/metrics"
	log "github.com/sirupsen/logrus"

	"github.com/nais/gemini/pkg/db"
)

const (
	CalendarPrivate = "private"
	CalendarPublic  = "public"
)

type Server struct {
	calendars   map[db.ID]*CalendarCache
	lastRequest map[string]time.Time // when was this calendar last requested?
	database    db.Database
	store       Store
}

func NewServer(database db.Database, store Store) *Server {
	srv := &Server{
		calendars:   make(map[db.ID]*CalendarCache),
		lastRequest: make(map[string]time.Time),
		database:    database,
		store:       store,
	}
	go srv.reportMetrics(30 * time.Second)
	return srv
}

// Update the "calendars currently in use" gauge.
// Run this as a Goroutine.
func (s *Server) reportMetrics(interval time.Duration) {
	t := time.NewTicker(interval)
	for range t.C {
		inUse := 0
		then := time.Now().Add(-24 * time.Hour)
		for _, lastRequestTime := range s.lastRequest {
			if lastRequestTime.After(then) {
				inUse++
			}
		}
		metrics.CalendarsInUse(inUse)
	}
}

// This page is what the user sees when they log in.
func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	templateParams := &TemplateParameters{}

	user, _ := r.Context().Value("user").(*db.User)
	if user != nil {
		// register user with calendar async fetcher
		s.store.Add(user)

		calendarInstance := s.store.Get(user.ID)
		if calendarInstance != nil {
			if calendarInstance.currentVersion != nil {
				templateParams.Size = len([]byte(calendarInstance.currentVersion.secretCalendar.Serialize()))
				templateParams.PublicSize = len([]byte(calendarInstance.currentVersion.publicCalendar.Serialize()))
			} else {
				templateParams.Error = "Venter på å bli synkronisert..."
			}

			templateParams.NextSync = calendarInstance.syncOptions.nextSync.Truncate(time.Second)
			templateParams.LastSync = calendarInstance.syncOptions.lastSync.Truncate(time.Second)
			templateParams.LastSuccess = calendarInstance.syncOptions.lastSuccess.Truncate(time.Second)
			templateParams.HasSuccess = !calendarInstance.syncOptions.lastSuccess.IsZero()
			templateParams.Disabled = calendarInstance.syncOptions.disabled
			if calendarInstance.syncOptions.err != nil {
				templateParams.Error = calendarInstance.syncOptions.err.Error()
			}
			templateParams.Authenticated = true
			templateParams.UserID = string(user.ID)
			templateParams.PublicID = user.ID.Public()
		}
	}

	err := tpl.Execute(w, templateParams)
	if err != nil {
		log.Errorf("BUG: template render error: %s", err)
	}
}

// Serve a user's public calendar with only busy/free information.
func (s *Server) PublicCalendar(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	publicID := chi.URLParam(r, "publicid")
	calendar := s.store.GetPublic(publicID)
	if calendar == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(w, "Calendar is not yet synchronized, please wait a few minutes.")
		metrics.Request(requestStart, CalendarPublic, false)
		return
	}

	s.lastRequest[publicID] = time.Now()

	payload := []byte(calendar.Serialize())
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(payload)

	metrics.Request(requestStart, CalendarPublic, true)
}

// Serve a user's secret calendar with all info intact.
func (s *Server) Calendar(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	userID := db.ID(chi.URLParam(r, "userid"))

	calendarInstance := s.store.Get(userID)

	if calendarInstance == nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintf(w, "Calendar does not exist. Please go to the index page and get your personal calendar URL.")
		metrics.Request(requestStart, CalendarPrivate, false)
		return
	}

	if calendarInstance.syncOptions.disabled {
		msg := "Your calendar is unavailable due to missing credentials. Please log in to Gemini again. For help, see #gemini on Slack."
		payload := []byte(ErrorCalendarWeek(msg).Serialize())
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
		metrics.Request(requestStart, CalendarPrivate, false)
		return
	}

	if calendarInstance.syncOptions.err != nil {
		log.Errorf("user's calendar is unavailable due to: %+v", calendarInstance.syncOptions.err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(w, "calendar is unavailable: %s", calendarInstance.syncOptions.err)
		metrics.Request(requestStart, CalendarPrivate, false)
		return
	}

	if calendarInstance.currentVersion == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(w, "Calendar is not yet synchronized, please wait a few minutes.")
		metrics.Request(requestStart, CalendarPrivate, false)
		return
	}

	s.lastRequest[string(userID)] = time.Now()

	payload := []byte(calendarInstance.currentVersion.secretCalendar.Serialize())
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	_, err := w.Write(payload)

	metrics.Request(requestStart, CalendarPrivate, err == nil)
}
