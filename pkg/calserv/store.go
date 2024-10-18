package calserv

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ics "github.com/arran4/golang-ical"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/nais/gemini/pkg/azure"
	"github.com/nais/gemini/pkg/db"
	"github.com/nais/gemini/pkg/metrics"
)

type Store interface {
	Add(user *db.User)
	Get(userid db.ID) *CalendarCache
}

// Asynchronous calendar fetcher.
type store struct {
	ctx      context.Context
	database db.Database
	lock     sync.Mutex
	queue    chan db.ID
	cache    map[db.ID]*CalendarCache
	timer    *time.Timer
	interval time.Duration
	lifetime time.Duration
	oauth    *oauth2.Config
}

// The translated version of a single user's calendar,
// together with metadata about its last synchronization.
type CalendarCache struct {
	userID      db.ID
	calendar    *ics.Calendar
	err         error
	disabled    bool // sync disabled for some reason, usually expired credentials
	lastSync    time.Time
	lastSuccess time.Time
	nextSync    time.Time
}

func NewStore(ctx context.Context, database db.Database, oauth *oauth2.Config, interval, lifetime time.Duration) Store {
	f := &store{
		cache:    make(map[db.ID]*CalendarCache),
		ctx:      ctx,
		database: database,
		interval: interval,
		lifetime: lifetime,
		oauth:    oauth,
		queue:    make(chan db.ID, 1024),
		timer:    time.NewTimer(interval),
	}
	go f.run()
	return f
}

func (f *store) run() {
	for {
		select {
		case <-f.ctx.Done():
			log.Debugf("Store shutting down")
			return
		case <-f.timer.C:
			f.timer.Reset(f.interval)
			f.fetchOutdated()
		case userid := <-f.queue:
			metrics.QueueSize(len(f.queue))
			f.fetch(userid)
		}
	}
}

func (f *store) fetchOutdated() {
	queued := 0
	f.lock.Lock()
	defer f.lock.Unlock()
	log.Debugf("Queueing all calendars for synchronization...")
	for i := range f.cache {
		if f.cache[i].disabled {
			continue
		}
		if f.cache[i].nextSync.After(time.Now()) {
			continue
		}
		f.cache[i].nextSync = time.Time{}
		f.queue <- f.cache[i].userID
		queued++
	}
	active, inactive := f.userCount()
	metrics.Users(active, inactive)
	metrics.QueueSize(len(f.queue))
	log.Debugf("All eligible calendars queued for synchronization (total of %d, vs %d active and %d inactive users).", queued, active, inactive)
}

func (f *store) fetch(userid db.ID) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.cache[userid] == nil {
		panic("BUG: fetching an unregistered calendar")
	}

	ctx, cancel := context.WithTimeout(f.ctx, time.Minute*4)
	defer cancel()

	user, err := f.database.GetUser(ctx, userid)
	if err != nil {
		f.cache[userid].err = err
		f.cache[userid].lastSync = time.Now()
		f.cache[userid].nextSync = time.Now().Add(1 * time.Minute)
		return
	}

	writeUser := func() {
		wctx, wcancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer wcancel()
		err = f.database.WriteUser(wctx, user)
		if err != nil {
			log.Error(err)
		}
	}

	// Auto-refreshes token when needed
	oldAccessToken := user.Token.AccessToken
	src := f.oauth.TokenSource(ctx, user.Token)
	client := oauth2.NewClient(ctx, src)

	log.Infof("Synchronizing calendar for user %s", user.Username)

	events, err := azure.GetCalendarEvents(client)
	metrics.Synchronizations(err)

	if err != nil {
		log.Errorf("synchronize %s: %s", user.Username, err)
		f.cache[userid].err = err
		f.cache[userid].lastSync = time.Now()
		f.cache[userid].nextSync = time.Now().Add(5 * time.Minute)

		// check for expired credentials
		var azureError *azure.ApiError
		if errors.As(err, &azureError) && azureError != nil && azureError.ErrorIdentifier == "invalid_grant" {
			f.cache[userid].err = azureError
			f.cache[userid].nextSync = time.Time{}
			f.cache[userid].disabled = true
			log.Warnf("account %q has invalid credentials and will be disabled until next login", user.Username)
			user.Token = nil
			writeUser()
		}

		return
	}

	user.Token, err = src.Token()
	if err == nil && oldAccessToken != user.Token.AccessToken {
		log.Infof("Token for user '%s' has been refreshed", user.Username)
	} else if err != nil {
		log.Error(err)
	}

	writeUser()

	now := time.Now()
	cal := Calendar(events)
	cal.SetLastModified(now)
	//cal.SetRefreshInterval()
	cal.SetName(user.Username)

	f.cache[userid].err = nil
	f.cache[userid].calendar = cal
	f.cache[userid].lastSync = now
	f.cache[userid].lastSuccess = now
	f.cache[userid].nextSync = now.Add(f.lifetime)
}

func (f *store) Add(user *db.User) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.cache[user.ID] != nil {
		return
	}
	log.Infof("Monitoring calendar for user %s", user.Username)
	f.cache[user.ID] = &CalendarCache{
		userID:   user.ID,
		err:      fmt.Errorf("not yet synchronized"),
		nextSync: time.Now(),
		disabled: user.Token == nil || len(user.Token.AccessToken) == 0,
	}
	f.timer.Reset(1 * time.Second)
	active, inactive := f.userCount()
	metrics.Users(active, inactive)
}

func (f *store) userCount() (active, inactive int) {
	for _, user := range f.cache {
		if user == nil {
			inactive++
			continue
		}
		if user.disabled {
			inactive++
		} else {
			active++
		}
	}
	return
}

func (f *store) Get(userid db.ID) *CalendarCache {
	return f.cache[userid]
}
