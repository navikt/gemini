package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/nais/gemini/pkg/authserv"
	"github.com/nais/gemini/pkg/calserv"
	"github.com/nais/gemini/pkg/config"
	"github.com/nais/gemini/pkg/db"
	"github.com/nais/gemini/pkg/version"
)

func main() {
	err := run()
	if err != nil {
		log.Errorf("fatal: %s\n", err)
		os.Exit(1)
	}
}

const syncInterval = time.Minute
const lifetime = 30 * time.Minute // cache calendars for this amount of time before fetching them again

func run() error {
	log.SetLevel(log.InfoLevel)
	log.Infof("GEMINI %s", version.Version())

	bt, err := version.BuildTime()
	if err == nil {
		log.Infof("Build time: %s", bt.String())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	cfg, err := config.FromEnvironment()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	log.Infof("Connecting to database...")
	database, err := setupDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	log.Infof("Database connection ready.")

	oauthconf := cfg.OAuthConfig()
	store := calserv.NewStore(ctx, database, oauthconf, syncInterval, lifetime)
	srv := calserv.NewServer(database, store)
	auth := authserv.NewServer(oauthconf, cfg.AzureClientID, database)
	validator := authserv.SessionIDMiddleware(database)
	router := setupRouter(srv, auth, validator)

	users, err := database.Users(ctx)
	if err != nil {
		return fmt.Errorf("load users from database: %w", err)
	}
	for _, user := range users {
		store.Add(user)
	}

	go func() {
		err := http.ListenAndServe(cfg.BindAddress, router)
		log.Errorf("http server has stopped: %s", err)
		cancel()
	}()

	go func() {
		err := http.ListenAndServe(cfg.MetricsBindAddress, promhttp.Handler())
		log.Errorf("metrics server has stopped: %s", err)
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
		r.HandleFunc("/logout", auth.Logout)
		r.HandleFunc("/callback", auth.Callback)
	})

	r.Route("/", func(r chi.Router) {
		r.Use(validator)
		r.HandleFunc("/", srv.Index)
	})

	r.HandleFunc("/calendar/{userid}", srv.Calendar)

	return r
}
