package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"

	"github.com/nais/gemini/pkg/authserv"
	"github.com/nais/gemini/pkg/calserv"
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

	clientid := os.Getenv("AZURE_APP_CLIENT_ID")
	dsn := os.Getenv("DATABASE_URL")
	bindAddress := os.Getenv("BIND_ADDRESS")
	metricsBindAddress := os.Getenv("METRICS_BIND_ADDRESS")
	connLimitStr := os.Getenv("DATABASE_CONNECTION_LIMIT")
	if len(bindAddress) == 0 {
		bindAddress = "127.0.0.1:3000"
	}
	if len(metricsBindAddress) == 0 {
		metricsBindAddress = "127.0.0.1:3001"
	}

	connLimit, err := strconv.Atoi(connLimitStr)
	if err != nil {
		connLimit = 0
	}

	oauthconf := &oauth2.Config{
		ClientID:     clientid,
		ClientSecret: os.Getenv("AZURE_APP_CLIENT_SECRET"),
		Endpoint:     microsoft.AzureADEndpoint(os.Getenv("AZURE_APP_TENANT_ID")),
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Scopes: []string{
			"Calendars.Read",
			"offline_access",
		},
	}

	// hack required for tiny cloud sql instances
	if len(dsn) > 0 && connLimit > 0 {
		dsn, err = dbURLWithConnectionLimit(dsn, connLimit)
		if err != nil {
			return fmt.Errorf("add connection limit to database url: %w", err)
		}
	}

	database, err := setupDatabase(dsn)
	if err != nil {
		return err
	}

	log.Infof("Database connection ready.")

	store := calserv.NewStore(ctx, database, oauthconf, syncInterval, lifetime)
	validator := authserv.SessionIDMiddleware(database)
	srv := calserv.NewServer(database, store)
	auth := authserv.NewServer(oauthconf, clientid, database)
	router := setupRouter(srv, auth, validator)

	users, err := database.Users(ctx)
	if err != nil {
		return fmt.Errorf("load users from database: %w", err)
	}
	for _, user := range users {
		store.Add(user)
	}

	go func() {
		err := http.ListenAndServe(bindAddress, router)
		log.Errorf("http server has stopped: %s", err)
		cancel()
	}()

	go func() {
		err := http.ListenAndServe(metricsBindAddress, promhttp.Handler())
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
		r.HandleFunc("/callback", auth.Callback)
	})

	r.Route("/", func(r chi.Router) {
		r.Use(validator)
		r.HandleFunc("/", srv.Index)
	})

	r.HandleFunc("/calendar/{userid}", srv.Calendar)

	return r
}

func dbURLWithConnectionLimit(dsn string, limit int) (string, error) {
	// hack to limit connections to database
	dburl, err := url.Parse(dsn)
	if err != nil {
		return dsn, err
	}
	q := dburl.Query()
	q.Add("pool_max_conns", strconv.Itoa(limit))
	dburl.RawQuery = q.Encode()
	return dburl.String(), nil
}
