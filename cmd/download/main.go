package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/nais/gemini/pkg/azure"
	"github.com/nais/gemini/pkg/config"
	"github.com/nais/gemini/pkg/db"
)

// Event downloader.
// Fetches API responses from Azure and puts all the events in a JSON file.
// Usage: go run cmd/download/main.go -email kim.tore.jensen@nav.no
func main() {
	err := run()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnvironment()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	err = database.Migrate(ctx)
	if err != nil {
		return err
	}

	email := flag.String("email", "", "email address of user (case sensitive, must exist in db)")
	outputPath := flag.String("output", "events.json", "json output filename")
	flag.Parse()
	if *email == "" {
		flag.Usage()
		return fmt.Errorf("need to specify user's email address")
	}

	userid, err := database.Lookup(ctx, *email)
	if err != nil {
		return err
	}

	user, err := database.GetUser(ctx, userid)
	if err != nil {
		return err
	}

	log.Infof("Downloading eligible events for %s...", user.Username)

	oauth := cfg.OAuthConfig()
	src := oauth.TokenSource(ctx, user.Token)
	client := oauth2.NewClient(ctx, src)
	events, err := azure.GetCalendarEvents(client)
	if err != nil {
		return err
	}

	log.Infof("Writing %s", *outputPath)

	output, err := os.Create(*outputPath)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(output)
	enc.SetIndent("", "  ")
	err = enc.Encode(events)
	if err != nil {
		return err
	}

	log.Infof("Finished.")
	return nil
}
