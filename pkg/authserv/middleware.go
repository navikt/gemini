package authserv

import (
	"context"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/nais/gemini/pkg/db"
)

// Checks that a session token is set in the cookie header, and that it exists in the database.
// Loads the correct user object from database and sets it on the http request context.
func SessionIDMiddleware(database db.Database) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			user, err := database.GetUser(r.Context(), db.ID(cookie.Value))
			if err != nil {
				if !errors.Is(err, db.ErrNotFound) {
					log.Errorf("database error: %s", err)
					http.Error(w, "database error; please try again later", http.StatusInternalServerError)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UserContextKey, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
