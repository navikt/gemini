package db

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v4"
	"golang.org/x/oauth2"
)

func (db *postgresDB) Users(ctx context.Context) ([]*User, error) {
	users := make([]*User, 0)
	query := `SELECT id, username, token FROM users`
	rows, err := db.timedQuery(ctx, query)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (db *postgresDB) WriteUser(ctx context.Context, user *User) error {
	query := `
INSERT INTO users (id, username, token)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE
SET username = EXCLUDED.username, token = EXCLUDED.token;
`

	token, err := json.Marshal(user.Token)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(ctx, query,
		user.ID,
		user.Username,
		token,
	)

	return err
}

func (db *postgresDB) Lookup(ctx context.Context, username string) (ID, error) {
	var id ID

	query := `SELECT id FROM users WHERE username = $1 LIMIT 1`
	rows, err := db.timedQuery(ctx, query, username)

	if err != nil {
		return id, err
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&id)
		return id, err
	}

	return id, ErrNotFound
}

func (db *postgresDB) GetUser(ctx context.Context, id ID) (*User, error) {
	query := `SELECT id, username, token FROM users WHERE id = $1 LIMIT 1`
	rows, err := db.timedQuery(ctx, query, id)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		return scanUser(rows)
	}

	return nil, ErrNotFound
}

func scanUser(row pgx.Row) (*User, error) {
	var token string
	user := &User{
		Token: &oauth2.Token{},
	}

	err := row.Scan(&user.ID, &user.Username, &token)
	if err != nil {
		return nil, err
	}

	r := strings.NewReader(token)
	err = json.NewDecoder(r).Decode(user.Token)
	if err != nil {
		return nil, err
	}

	return user, nil
}
