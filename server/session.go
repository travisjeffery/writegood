package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v4"
)

type SessionStore struct {
	conn     *pgx.Conn
	codecs   []securecookie.Codec
	opts     sessions.Options
	initOnce sync.Once
}

// Get returns a cached session.
func (s *SessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	s.init()

	if err := s.conn.QueryRow(context.Background(), `select key, user_id, created, updated, expires from sessions where key = $1`, name).Scan(); err != nil {
		return nil, err
	}
}

// New creates and returns a new session, should never return a nil session.
func (s *SessionStore) New(r *http.Request, name string) (*sessions.Session, error) {
	s.init()

	cookieSession := sessions.NewSession(s, name)
	cookieSession.Options = &s.opts
	cookieSession.IsNew = true

	c, err := r.Cookie(name)
	if err != nil {
		return cookieSession, err
	}
	err = securecookie.DecodeMulti(name, c.Value, &cookieSession.ID, s.codecs...)
	if err != nil {
		return cookieSession, err
	}

	var persistedSession Session
	if err := s.conn.QueryRow(context.Background(), `select key, user_id, created, updated, expires from sessions where key = $1`, name).
		Scan(&persistedSession.Key, &persistedSession.UserID, &persistedSession.Created, &persistedSession.Updated, &persistedSession.Expires); err != nil {
		return cookieSession, err
	}
	if err == nil {
		cookieSession.IsNew = false
		_ = securecookie.DecodeMulti(cookieSession.Name(), persistedSession.Data, &cookieSession.Values, s.codecs...)
	}
	if err == pgx.ErrNoRows {
		err = nil
	}

	return cookieSession, err
}

// Save persists the session.
func (s *SessionStore) Save(r *http.Request, w http.ResponseWriter, cookieSession *sessions.Session) error {
	s.init()

	encoded, err := securecookie.EncodeMulti(cookieSession.Name(), cookieSession.Values, s.codecs...)
	if err != nil {
		return err
	}

	created := cookieSession.Values["created"].(time.Time)
	expires := cookieSession.Values["expires"].(time.Time)

	return nil
}

func (s *SessionStore) init() {
	s.initOnce.Do(func() {
		s.opts.Secure = true
		s.opts.HttpOnly = true
	})
}

type Session struct {
	Key     string
	UserID  int
	Data    string
	Created time.Time
	Updated time.Time
	Expires time.Time
}
