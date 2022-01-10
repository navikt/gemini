package calserv

import (
	"net/http"

	"github.com/ambientsound/gemini/pkg/azure"
)

type Server struct {
	Events azure.Events
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cal := Calendar(s.Events)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(cal.Serialize()))
}

var _ http.Handler = &Server{}
