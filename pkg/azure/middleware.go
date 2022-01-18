package azure

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
)

type Claims struct {
	UserID    string   `json:"upn"`
	Audience  string   `json:"aud"`
	IssuedAt  UnixTime `json:"iat"`
	NotBefore UnixTime `json:"nbf"`
	Expiry    UnixTime `json:"exp"`
}

type UnixTime struct {
	time.Time
}

func (t *UnixTime) UnmarshalJSON(b []byte) error {
	secs, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return err
	}
	t.Time = time.Unix(secs, 0)
	return nil
}

func (c *Claims) Verify(audience string) error {
	now := time.Now()
	if c.NotBefore.After(now) {
		return fmt.Errorf("token is not valid yet")
	}
	if c.IssuedAt.After(now) {
		return fmt.Errorf("token is issued in the future")
	}
	if c.Expiry.Before(now) {
		return fmt.Errorf("token has expired")
	}
	if c.Audience != audience {
		return fmt.Errorf("token has wrong audience")
	}
	return nil
}

// Validates issuer, validity time, and audience.
// This program is designed to run behind Wonderwall, and as such, it does not validate the token cryptography.
func TokenValidatorMiddleware(audience string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			werr := func(err error) {
				w.Write([]byte(err.Error()))
			}

			const bearer = "Bearer "
			authHeader := r.Header.Get("authorization")
			if len(authHeader) <= len(bearer) || !strings.HasPrefix(authHeader, bearer) {
				http.Redirect(w, r, "/oauth2/login", http.StatusTemporaryRedirect)
				return
			}

			tokenString := authHeader[len(bearer):]

			token, err := jwt.ParseString(
				tokenString,
				jwt.WithAudience(audience),
				jwt.WithValidate(true),
			)

			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				werr(err)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "token", tokenString)
			ctx = context.WithValue(ctx, "claims", token)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
