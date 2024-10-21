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
	ctx         context.Context
	database    db.Database
	lock        sync.Mutex
	fetchQueue  chan db.ID
	updateQueue chan CalendarInstance
	errorQueue  chan CalendarSyncError
	cache       map[db.ID]*CalendarCache
	timer       *time.Timer
	interval    time.Duration
	lifetime    time.Duration
	oauth       *oauth2.Config
}

// A single instance of a converted calendar
type CalendarInstance struct {
	userID   db.ID
	calendar ics.Calendar
	//startSyncTime time.Time
	//endSyncTime   time.Time
}

type CalendarSyncError struct {
	userID db.ID
	err    error
}

var ErrCredentialsExpired = fmt.Errorf("credentials no longer valid, please re-authenticate")

// Metadata required to sync at certain intervals
type SynchronizationOptions struct {
	err         error
	syncing     bool
	disabled    bool
	lastSync    time.Time
	lastSuccess time.Time
	nextSync    time.Time
}

// The translated version of a single user's calendar,
// together with metadata about its last synchronization.
type CalendarCache struct {
	currentVersion *CalendarInstance
	syncOptions    SynchronizationOptions
}

func NewStore(ctx context.Context, database db.Database, oauth *oauth2.Config, interval, lifetime time.Duration) Store {
	f := &store{
		cache:       make(map[db.ID]*CalendarCache),
		ctx:         ctx,
		database:    database,
		interval:    interval,
		lifetime:    lifetime,
		oauth:       oauth,
		fetchQueue:  make(chan db.ID, 1024),
		updateQueue: make(chan CalendarInstance, 1024),
		errorQueue:  make(chan CalendarSyncError, 1024),
		timer:       time.NewTimer(interval),
	}
	go f.run()
	return f
}

func (f *store) run() {
	jobs := 0

	for {
		select {
		case <-f.ctx.Done():
			log.Debugf("Store shutting down")
			return

		case <-f.timer.C:
			f.timer.Reset(f.interval)
			f.fetchOutdated()

		case calendarInstance := <-f.updateQueue:
			f.store(calendarInstance)
			jobs--

		case syncError := <-f.errorQueue:
			f.handleSyncError(syncError)
			jobs--

		// Fetch calendars and put the results on a queue.
		// This takes a long time, and may run concurrently.
		case userid := <-f.fetchQueue:
			jobs++
			go f.fetchAndEnqueue(userid)
		}

		metrics.QueueSize(jobs)
	}
}

// This function is meant to run concurrently.
func (f *store) fetchAndEnqueue(userid db.ID) {
	calendarInstance, err := f.fetch(userid)

	if err == nil {
		f.updateQueue <- CalendarInstance{
			userID:   userid,
			calendar: *calendarInstance,
		}
	} else {
		f.errorQueue <- CalendarSyncError{
			userID: userid,
			err:    err,
		}
	}
}

func (f *store) fetchOutdated() {
	f.lock.Lock()
	defer f.lock.Unlock()
	queued := 0

	for i := range f.cache {
		if f.cache[i].syncOptions.nextSync.After(time.Now()) {
			continue
		}
		if f.cache[i].syncOptions.syncing || f.cache[i].syncOptions.disabled {
			continue
		}
		f.fetchQueue <- i
		queued++
	}

	active, inactive := f.userCount()
	metrics.Users(active, inactive)
	log.Debugf("Eligible calendars queued for synchronization (total of %d, vs %d active and %d inactive users).", queued, active, inactive)
}

// Store a user's calendar in the cache.
func (f *store) store(instance CalendarInstance) {
	f.lock.Lock()
	defer f.lock.Unlock()

	entry := f.cache[instance.userID]

	now := time.Now()
	entry.currentVersion = &instance
	entry.syncOptions.err = nil
	entry.syncOptions.syncing = false
	entry.syncOptions.lastSync = now
	entry.syncOptions.lastSuccess = now
	entry.syncOptions.nextSync = now.Add(f.lifetime)

	f.cache[instance.userID] = entry
}

// Make sure synchronization errors update the cache.
func (f *store) handleSyncError(error CalendarSyncError) {
	const retryInterval = 5 * time.Minute
	f.lock.Lock()
	defer f.lock.Unlock()

	entry := f.cache[error.userID]

	now := time.Now()
	entry.syncOptions.err = error.err
	entry.syncOptions.lastSync = now
	entry.syncOptions.syncing = false

	if errors.Is(error.err, ErrCredentialsExpired) {
		// disable syncing users with expired credentials
		entry.syncOptions.disabled = true
	} else {
		entry.syncOptions.nextSync = now.Add(retryInterval)
	}

	f.cache[error.userID] = entry
}

func (f *store) fetch(userid db.ID) (*ics.Calendar, error) {
	const timeout = 4 * time.Minute
	const databaseTimeoutNextRetry = 5 * time.Second

	f.lock.Lock()
	f.cache[userid].syncOptions.syncing = true
	f.lock.Unlock()

	ctx, cancel := context.WithTimeout(f.ctx, timeout)
	defer cancel()

	user, err := f.database.GetUser(ctx, userid)
	if err != nil {
		return nil, err
	}

	writeUser := func() error {
		wctx, wcancel := context.WithTimeout(context.Background(), databaseTimeoutNextRetry)
		defer wcancel()
		return f.database.WriteUser(wctx, user)
	}

	// Auto-refreshes token when needed
	oldAccessToken := user.Token.AccessToken
	src := f.oauth.TokenSource(ctx, user.Token)
	client := oauth2.NewClient(ctx, src)

	log.Infof("Synchronizing calendar for user %s", user.Username)

	t := time.Now()
	events, err := azure.GetCalendarEvents(client)
	metrics.Synchronizations(t, err)

	if err != nil {
		// check for expired credentials
		var azureError *azure.ApiError
		if errors.As(err, &azureError) && azureError != nil && azureError.ErrorIdentifier == "invalid_grant" {
			user.Token = nil
			err = writeUser()

			// database error gets priority
			if err == nil {
				return nil, ErrCredentialsExpired
			}
		}

		return nil, err
	}

	log.Infof("Calendar synchronization for user %s finished in %s", user.Username, time.Since(t))

	user.Token, err = src.Token()
	if err == nil && oldAccessToken != user.Token.AccessToken {
		log.Infof("Token for user '%s' has been refreshed", user.Username)
	}

	err = writeUser()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	cal := Calendar(events)
	cal.SetLastModified(now)
	cal.SetName(user.Username)

	return cal, nil
}

func (f *store) Add(user *db.User) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.cache[user.ID] != nil {
		return
	}

	f.cache[user.ID] = &CalendarCache{
		currentVersion: nil,
		syncOptions: SynchronizationOptions{
			nextSync: time.Now(),
		},
	}

	if user.Token != nil && len(user.Token.AccessToken) > 0 {
		log.Infof("Calendar for %s is now monitored", user.Username)
	} else {
		f.cache[user.ID].syncOptions.disabled = true
		log.Warnf("Calendar for %s is not monitored due to missing credentials", user.Username)
	}

	// Schedule next sync now
	f.timer.Reset(1 * time.Second)

	active, inactive := f.userCount()
	metrics.Users(active, inactive)
}

func (f *store) userCount() (active, inactive int) {
	for _, calendarInstance := range f.cache {
		if calendarInstance == nil || calendarInstance.currentVersion == nil {
			inactive++
			continue
		}
		if calendarInstance.syncOptions.disabled {
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
