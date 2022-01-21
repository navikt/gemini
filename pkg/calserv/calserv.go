package calserv

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ambientsound/gemini/pkg/db"
	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
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
	user, ok := r.Context().Value("user").(*db.User)
	if !ok || user == nil {
		log.Errorf("BUG: index handler called, but no user provided")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// register user ID with calendar async fetcher
	s.store.Add(user.ID)

	w.Header().Set("content-type", "text/html")
	fmt.Fprintf(w, `Please copy your <a href="/calendar/%s">personal calendar link</a> and subscribe to it in your calendar application.`, user.ID)
}

func (s *Server) SetCalendar(userid db.ID, calendar *CalendarCache) {
	s.calendars[userid] = calendar
}

func (s *Server) Calendar(w http.ResponseWriter, r *http.Request) {
	userID := db.ID(chi.URLParam(r, "userid"))

	cache := s.store.Get(userID)

	if cache == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Calendar does not exist. Please go to the index page and get your personal calendar URL.")
		return
	}

	if cache.err != nil {
		log.Error(cache.err)
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "calendar is unavailable: %s", cache.err)
		return
	}

	payload := []byte(cache.calendar.Serialize())
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	w.Write(payload)
}
