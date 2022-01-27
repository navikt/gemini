package db

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type inMemoryDatabase struct {
	table map[ID]*User
}

func (i *inMemoryDatabase) WriteUser(_ context.Context, user *User) error {
	log.Debugf("Stored user '%s' in database", user.Username)
	i.table[user.ID] = user
	return nil
}

func (i *inMemoryDatabase) Lookup(_ context.Context, username string) (ID, error) {
	log.Debugf("Lookup '%s' in database", username)
	for id, user := range i.table {
		if user.Username == username {
			return id, nil
		}
	}
	return "", ErrUsernameNotFound
}

func (i *inMemoryDatabase) GetUser(_ context.Context, id ID) (*User, error) {
	log.Debugf("Retrieving user ID '%s' from database", id)
	user, found := i.table[id]
	if !found {
		return nil, ErrUserIDNotFound
	}
	return user, nil
}

func (i *inMemoryDatabase) Migrate(_ context.Context) error {
	log.Debugf("Database migration not needed for in-memory database")
	return nil
}

var _ Database = &inMemoryDatabase{}

func NewInMemoryDatabase() Database {
	return &inMemoryDatabase{
		table: make(map[ID]*User),
	}
}
