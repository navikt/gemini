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
	Size          int
	LastSync      time.Time
	LastSuccess   time.Time
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
