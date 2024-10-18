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

type Server struct {
	calendars map[db.ID]*CalendarCache
	database  db.Database
	store     Store
}

func NewServer(database db.Database, store Store) *Server {
	return &Server{
		calendars: make(map[db.ID]*CalendarCache),
		database:  database,
		store:     store,
	}
}

func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	templateParams := &TemplateParameters{}

	user, _ := r.Context().Value("user").(*db.User)
	if user != nil {
		// register user with calendar async fetcher
		s.store.Add(user)

		cache := s.store.Get(user.ID)
		if cache != nil && cache.calendar != nil {
			templateParams.Size = len([]byte(cache.calendar.Serialize()))
			templateParams.NextSync = cache.nextSync.Truncate(time.Second)
			templateParams.LastSync = cache.lastSync.Truncate(time.Second)
			templateParams.LastSuccess = cache.lastSuccess.Truncate(time.Second)
			templateParams.Disabled = cache.disabled
			if cache.err != nil {
				templateParams.Error = cache.err.Error()
			}
		}

		templateParams.Authenticated = true
		templateParams.UserID = string(user.ID)
	}

	err := tpl.Execute(w, templateParams)
	if err != nil {
		log.Errorf("BUG: template render error: %s", err)
	}
}

func (s *Server) SetCalendar(userid db.ID, calendar *CalendarCache) {
	s.calendars[userid] = calendar
}

func (s *Server) Calendar(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	userID := db.ID(chi.URLParam(r, "userid"))

	cache := s.store.Get(userID)

	if cache == nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintf(w, "Calendar does not exist. Please go to the index page and get your personal calendar URL.")
		metrics.Request(requestStart, false)
		return
	}

	if cache.err != nil {
		log.Error(cache.err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(w, "calendar is unavailable: %s", cache.err)
		metrics.Request(requestStart, false)
		return
	}

	payload := []byte(cache.calendar.Serialize())
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	_, err := w.Write(payload)

	metrics.Request(requestStart, err == nil)
}
