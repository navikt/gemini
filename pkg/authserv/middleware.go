package authserv

import (
	"context"
	"net/http"

	"github.com/nais/gemini/pkg/db"
	log "github.com/sirupsen/logrus"
)

// Checks that a session token is set in the cookie header, and that it exists in the database.
// Loads the correct user object from database and sets it on the http request context.
func SessionIDMiddleware(database db.Database) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				http.Redirect(w, r, "/oauth2/login", http.StatusTemporaryRedirect)
				return
			}

			user, err := database.GetUser(r.Context(), db.ID(cookie.Value))
			if err != nil {
				if err != db.ErrNotFound {
					log.Errorf("database error: %s", err)
					http.Error(w, "database error; please try again later", http.StatusInternalServerError)
					return
				}
				http.Redirect(w, r, "/oauth2/login", http.StatusTemporaryRedirect)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "user", user)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
