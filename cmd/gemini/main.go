package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi"
	"github.com/nais/gemini/pkg/authserv"
	"github.com/nais/gemini/pkg/calserv"
	"github.com/nais/gemini/pkg/db"
	"github.com/nais/gemini/pkg/version"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

func main() {
	err := run()
	if err != nil {
		log.Errorf("fatal: %s\n", err)
		os.Exit(1)
	}
}

const syncInterval = time.Minute
const lifetime = time.Hour

func run() error {
	log.SetLevel(log.TraceLevel)
	log.Infof("GEMINI %s", version.Version())

	bt, err := version.BuildTime()
	if err == nil {
		log.Infof("Build time: %s", bt.String())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	clientid := os.Getenv("CLIENT_ID")
	iss := os.Getenv("ISSUER")
	dsn := os.Getenv("DSN")

	oauthconf := &oauth2.Config{
		ClientID:     clientid,
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		Endpoint:     microsoft.AzureADEndpoint(os.Getenv("TENANT_ID")),
		RedirectURL:  "http://localhost:3000/oauth2/callback",
		Scopes: []string{
			"Calendars.Read",
			"offline_access",
		},
	}

	database, err := setupDatabase(dsn)
	if err != nil {
		return err
	}

	log.Infof("Database connection ready.")

	store := calserv.NewStore(ctx, database, oauthconf, syncInterval, lifetime)
	validator := authserv.SessionIDMiddleware(database)
	srv := calserv.NewServer(database, store)
	auth := authserv.NewServer(oauthconf, iss, clientid, database)
	router := setupRouter(srv, auth, validator)

	go func() {
		err := http.ListenAndServe("127.0.0.1:3000", router)
		log.Errorf("http server has stopped: %s", err)
		cancel()
	}()

	log.Infof("Serving requests...")

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

func setupDatabase(dsn string) (db.Database, error) {
	if len(dsn) == 0 {
		return db.NewInMemoryDatabase(), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	database, err := db.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	err = database.Migrate(ctx)
	if err != nil {
		return nil, err
	}

	return database, nil
}

func setupRouter(srv *calserv.Server, auth *authserv.Server, validator func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Route("/oauth2", func(r chi.Router) {
		r.HandleFunc("/login", auth.Login)
		r.HandleFunc("/callback", auth.Callback)
	})

	r.Route("/", func(r chi.Router) {
		r.Use(validator)
		r.HandleFunc("/", srv.Index)
	})

	r.HandleFunc("/calendar/{userid}", srv.Calendar)

	return r
}
