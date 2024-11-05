package authserv

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/jwt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/nais/gemini/pkg/db"
)

const (
	StateCookieName   = "gemini-oauth2-state"
	SessionCookieName = "gemini-session-id"
)

type Server struct {
	audience string
	cfg      *oauth2.Config
	database db.Database
}

type contextKey struct {
	Key string
}

var UserContextKey = contextKey{Key: "user"}

func NewServer(cfg *oauth2.Config, audience string, database db.Database) *Server {
	return &Server{
		cfg:      cfg,
		database: database,
		audience: audience,
	}
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	state, err := uuid.NewRandom()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	uri := s.cfg.AuthCodeURL(state.String())

	cookie := &http.Cookie{
		Name:     StateCookieName,
		Value:    state.String(),
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		//Secure:     false,
		//SameSite:   http.SameSiteDefaultMode,
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, uri, http.StatusTemporaryRedirect)
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Path:     "/",
		Expires:  time.Time{},
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) Callback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(StateCookieName)
	if err != nil {
		http.Error(w, "cannot authorize without state cookie", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != cookie.Value {
		http.Error(w, "wrong state cookie", http.StatusBadRequest)
		return
	}

	token, err := s.cfg.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Delete state cookie
	cookie = &http.Cookie{
		Name:     StateCookieName,
		Path:     "/",
		Expires:  time.Time{},
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)

	accessToken, err := jwt.ParseString(token.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("parse access token: %s", err), http.StatusUnauthorized)
		return
	}

	username, ok := accessToken.Get("upn")
	if !ok {
		http.Error(w, fmt.Sprintf("get username from token: %s", err), http.StatusUnauthorized)
		return
	}

	sessionID, err := s.database.Lookup(r.Context(), username.(string))
	if err != nil {
		// If user is not found, create a new one.
		sessionID, err = db.NewID()
		if err != nil {
			log.Errorf("unable to generate new ID for user: %s", err)
			http.Error(w, "internal error, please try again later", http.StatusInternalServerError)
			return
		}
		user := &db.User{
			ID:       sessionID,
			Username: username.(string),
			Token:    token,
		}
		err = s.database.WriteUser(r.Context(), user)
		if err != nil {
			log.Errorf("unable to store user in database: %s", err)
			http.Error(w, "internal error, please try again later", http.StatusInternalServerError)
			return
		}
	} else {
		// Update existing user's access token.
		user, err := s.database.GetUser(r.Context(), sessionID)
		if err != nil {
			log.Errorf("unable to get user from database: %s", err)
			http.Error(w, "internal error, please try again later", http.StatusInternalServerError)
			return
		}

		user.Token = token
		err = s.database.WriteUser(r.Context(), user)
		if err != nil {
			log.Errorf("unable to store user in database: %s", err)
			http.Error(w, "internal error, please try again later", http.StatusInternalServerError)
			return
		}
	}

	cookie = &http.Cookie{
		Name:     SessionCookieName,
		Value:    string(sessionID),
		Path:     "/",
		Expires:  time.Now().Add(1440 * time.Hour),
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
