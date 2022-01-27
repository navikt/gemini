package calserv

import (
	"context"
	"fmt"
	"sync"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/nais/gemini/pkg/azure"
	"github.com/nais/gemini/pkg/db"
	"github.com/nais/gemini/pkg/metrics"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Store interface {
	Add(user *db.User)
	Get(userid db.ID) *CalendarCache
}

type store struct {
	ctx      context.Context
	database db.Database
	lock     sync.Mutex
	queue    chan db.ID
	cache    map[db.ID]*CalendarCache
	ticker   *time.Ticker
	interval time.Duration
	lifetime time.Duration
	oauth    *oauth2.Config
}

type CalendarCache struct {
	userID      db.ID
	calendar    *ics.Calendar
	err         error
	lastSync    time.Time
	lastSuccess time.Time
	nextSync    time.Time
}

func NewStore(ctx context.Context, database db.Database, oauth *oauth2.Config, interval, lifetime time.Duration) *store {
	f := &store{
		cache:    make(map[db.ID]*CalendarCache),
		ctx:      ctx,
		database: database,
		interval: interval,
		lifetime: lifetime,
		oauth:    oauth,
		queue:    make(chan db.ID, 1024),
		ticker:   time.NewTicker(interval),
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
		case <-f.ticker.C:
			f.ticker.Reset(f.interval)
			f.fetchOutdated()
		case userid := <-f.queue:
			go f.fetch(userid)
		}
	}
}

func (f *store) fetchOutdated() {
	f.lock.Lock()
	defer f.lock.Unlock()
	log.Debugf("Synchronizing all calendars...")
	for i := range f.cache {
		if f.cache[i].nextSync.After(time.Now()) {
			continue
		}
		f.cache[i].nextSync = time.Time{}
		f.queue <- f.cache[i].userID
	}
	log.Debugf("Finished calendar synchronization.")
}

func (f *store) fetch(userid db.ID) {
	if f.cache[userid] == nil {
		panic("BUG: fetching an unregistered calendar")
	}

	ctx, cancel := context.WithTimeout(f.ctx, time.Minute*2)
	defer cancel()

	user, err := f.database.GetUser(ctx, userid)
	if err != nil {
		f.cache[userid].err = err
		f.cache[userid].lastSync = time.Now()
		f.cache[userid].nextSync = time.Now().Add(1 * time.Minute)
		return
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
		return
	}

	user.Token, err = src.Token()
	if oldAccessToken != user.Token.AccessToken {
		log.Infof("Token for user '%s' has been refreshed", user.Username)
	}
	if err == nil {
		err = f.database.WriteUser(ctx, user)
		if err != nil {
			err = fmt.Errorf("could not write updated token to database: %w", err)
		}
	}

	if err != nil {
		log.Error(err)
	}

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
	}
	f.ticker.Reset(1 * time.Second)
	metrics.Users.Set(float64(len(f.cache)))
}

func (f *store) Get(userid db.ID) *CalendarCache {
	return f.cache[userid]
}
