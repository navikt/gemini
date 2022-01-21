package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/oauth2"
)

type ID string

type User struct {
	ID       ID
	Username string
	Token    *oauth2.Token
}

type Database interface {
	Lookup(ctx context.Context, username string) (ID, error)
	GetUser(ctx context.Context, id ID) (*User, error)
	WriteUser(ctx context.Context, user *User) error
}

var (
	ErrUsernameNotFound = errors.New("user not cached in database")
	ErrUserIDNotFound   = errors.New("user id not found in database")
)

func NewID() (ID, error) {
	buf := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}
	return ID(base64.URLEncoding.EncodeToString(buf)), nil
}
