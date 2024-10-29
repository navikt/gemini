package calserv

import (
	"fmt"
	"html/template"
	"time"

	"github.com/nais/gemini/html"
)

type TemplateParameters struct {
	Authenticated bool
	UserID        string
	PublicID      string
	Size          int
	PublicSize    int
	Disabled      bool
	NextSync      time.Time
	LastSync      time.Time
	LastSuccess   time.Time
	HasSuccess    bool
	Error         string
}

var tpl *template.Template

func init() {
	var err error
	tpl, err = template.New("index.html").ParseFS(html.FS, "index.html")
	if err != nil {
		panic(fmt.Sprintf("BUG: template loader error: %s", err))
	}
}
