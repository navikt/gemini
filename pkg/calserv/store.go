package calserv

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ambientsound/gemini/pkg/azure"
	"github.com/ambientsound/gemini/pkg/db"
	ics "github.com/arran4/golang-ical"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Store interface {
	Add(userid db.ID)
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
}

type CalendarCache struct {
	userID      db.ID
	calendar    *ics.Calendar
	err         error
	lastSync    time.Time
	lastSuccess time.Time
	nextSync    time.Time
}

func NewStore(ctx context.Context, database db.Database, interval, lifetime time.Duration) *store {
	f := &store{
		ctx:      ctx,
		database: database,
		queue:    make(chan db.ID, 1024),
		cache:    make(map[db.ID]*CalendarCache),
		ticker:   time.NewTicker(interval),
		interval: interval,
		lifetime: lifetime,
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
	for i := range f.cache {
		if f.cache[i].nextSync.After(time.Now()) {
			continue
		}
		f.cache[i].nextSync = time.Time{}
		f.queue <- f.cache[i].userID
	}
}

func (f *store) fetch(userid db.ID) {
	if f.cache[userid] == nil {
		panic("BUG: fetching an unregistered calendar")
	}

	ctx, cancel := context.WithTimeout(f.ctx, time.Second)
	defer cancel()

	user, err := f.database.GetUser(ctx, userid)
	if err != nil {
		f.cache[userid].err = err
		f.cache[userid].lastSync = time.Now()
		f.cache[userid].nextSync = time.Now().Add(1 * time.Minute)
		return
	}

	token := &oauth2.Token{
		AccessToken: user.Token,
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	events, err := azure.GetCalendarEvents(client)
	if err != nil {
		f.cache[userid].err = err
		f.cache[userid].lastSync = time.Now()
		f.cache[userid].nextSync = time.Now().Add(5 * time.Minute)
		return
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

func (f *store) Add(userid db.ID) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.cache[userid] != nil {
		return
	}
	f.cache[userid] = &CalendarCache{
		userID:   userid,
		err:      fmt.Errorf("not yet synchronized"),
		nextSync: time.Now(),
	}
	f.ticker.Reset(1 * time.Second)
}

func (f *store) Get(userid db.ID) *CalendarCache {
	return f.cache[userid]
}
