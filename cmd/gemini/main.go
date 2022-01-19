package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ambientsound/gemini/pkg/azure"
	"github.com/ambientsound/gemini/pkg/calserv"
	"github.com/ambientsound/gemini/pkg/db"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := run()
	if err != nil {
		log.Errorf("fatal: %s\n", err)
		os.Exit(1)
	}
}

const audience = "00000003-0000-0000-c000-000000000000"
const syncInterval = time.Minute
const lifetime = time.Hour

func run() error {
	log.SetLevel(log.TraceLevel)
	log.Infof("GEMINI starting up - Office365 to iCal")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	database := db.NewInMemoryDatabase()
	store := calserv.NewStore(ctx, database, syncInterval, lifetime)
	validator := azure.TokenValidatorMiddleware(audience)
	srv := calserv.NewServer(database, store)
	router := setupRouter(srv, validator)

	go func() {
		err := http.ListenAndServe("0.0.0.0:8080", router)
		log.Errorf("http server has stopped: %s", err)
		cancel()
	}()

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			break
		case sig := <-sigs:
			log.Infof("received signal %s; shutting down...", sig)
			cancel()
		default:
		}
	}

	return nil
}

func setupRouter(srv *calserv.Server, validator func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.HandleFunc("/oauth2", func(w http.ResponseWriter, r *http.Request) {
		//goland:noinspection GoErrorStringFormat
		err := fmt.Errorf("oauth2 authentication not implemented in GEMINI; please run this program with the Wonderwall proxy in front.")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(err.Error()))
	})

	//r.With(validator).HandleFunc("/",srv.Index)
	r.Route("/", func(r chi.Router) {
		r.Use(validator)
		r.HandleFunc("/", srv.Index)
	})

	r.HandleFunc("/calendar/{userid}", srv.Calendar)

	return r
}
