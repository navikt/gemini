package calserv

import (
	"net/http"

	ics "github.com/arran4/golang-ical"
)

type Server struct {
	calendar *ics.Calendar
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//payload := []byte(s.calendar.Serialize())
	//w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	s.calendar.SerializeTo(w)
	//w.Write(payload)
}

func (s *Server) SetCalendar(cal *ics.Calendar) {
	s.calendar = cal
}

var _ http.Handler = &Server{}
