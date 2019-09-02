package session

import (
	"context"
	"encoding/base32"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v4"
)

type Store struct {
	Conn     *pgx.Conn
	Codecs   []securecookie.Codec
	Opts     *sessions.Options
	initOnce sync.Once
}

// Get returns a cached session.
func (s *Store) Get(r *http.Request, name string) (*sessions.Session, error) {
	s.init()
	return sessions.GetRegistry(r).Get(s, name)
}

// New creates and returns a new session, should never return a nil session.
func (s *Store) New(r *http.Request, name string) (*sessions.Session, error) {
	s.init()

	defer func() { s.maxAge(s.Opts.MaxAge) }()

	opts := *s.Opts
	cookieSession := sessions.NewSession(s, name)
	cookieSession.Options = &opts
	cookieSession.IsNew = true

	c, err := r.Cookie(name)
	if err != nil {
		return cookieSession, err
	}
	err = securecookie.DecodeMulti(name, c.Value, &cookieSession.ID, s.Codecs...)
	if err != nil {
		return cookieSession, err
	}

	persistedSession, err := s.get(cookieSession.ID)
	if err == nil {
		cookieSession.IsNew = false
		_ = securecookie.DecodeMulti(cookieSession.Name(), persistedSession.Data, &cookieSession.Values, s.Codecs...)
	}
	if err == pgx.ErrNoRows {
		err = nil
	}

	return cookieSession, err
}

// Save persists the session.
func (s *Store) Save(r *http.Request, w http.ResponseWriter, cookieSession *sessions.Session) error {
	s.init()

	if cookieSession.Options.MaxAge < 0 {
		if err := s.delete(cookieSession.ID); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(cookieSession.Name(), "", cookieSession.Options))
	}

	if cookieSession.ID == "" {
		cookieSession.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(32),
			), "=",
		)
	}

	if err := s.save(cookieSession); err != nil {
		return err
	}

	encoded, err := securecookie.EncodeMulti(cookieSession.Name(), cookieSession.ID, s.Codecs...)
	if err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(cookieSession.Name(), encoded, cookieSession.Options))

	return nil
}

// save persists cookie session.
func (s *Store) save(cookieSession *sessions.Session) error {
	persistedSession, err := s.persistedSessionFromCookieSession(cookieSession)
	if err != nil {
		return err
	}

	if cookieSession.IsNew {
		return s.insert(persistedSession)
	}

	return s.update(persistedSession)
}

// insert persisted session.
func (s *Store) insert(persistedSession PersistedSession) error {
	_, err := s.Conn.Exec(
		context.Background(),
		`insert into sessions (key, user_id, data, created, updated, expires) values ($1, $2, $3, $4, $5, $6)`,
		persistedSession.Key,
		persistedSession.UserID,
		persistedSession.Data,
		persistedSession.Created,
		persistedSession.Updated,
		persistedSession.Expires,
	)
	return err
}

// update persisted session.
func (s *Store) update(persistedSession PersistedSession) error {
	_, err := s.Conn.Exec(
		context.Background(),
		`update sessions set data = $1, updated = $2, expires = $3 where key = $4`,
		persistedSession.Data,
		persistedSession.Updated,
		persistedSession.Expires,
		persistedSession.Key,
	)
	return err
}

// delete persisted session.
func (s *Store) delete(key string) error {
	_, err := s.Conn.Exec(
		context.Background(),
		`delete from session where key = $1`,
		key,
	)
	return err
}

// get returns the persisted session.
func (s *Store) get(key string) (persistedSession PersistedSession, err error) {
	err = s.Conn.QueryRow(context.Background(), `select key, user_id, created, updated, expires from sessions where key = $1`, key).
		Scan(&persistedSession.Key, &persistedSession.UserID, &persistedSession.Created, &persistedSession.Updated, &persistedSession.Expires)
	return
}

// maxAge sets the max age for the store. You can delete invidual sessions by setting the age to -1.
func (s *Store) maxAge(age int) {
	for _, c := range s.Codecs {
		codec, ok := c.(*securecookie.SecureCookie)
		if !ok {
			continue
		}
		codec.MaxAge(age)
	}
}

func (s *Store) persistedSessionFromCookieSession(cookieSession *sessions.Session) (persistedSession PersistedSession, err error) {
	encoded, err := securecookie.EncodeMulti(cookieSession.Name(), cookieSession.Values, s.Codecs...)
	if err != nil {
		return persistedSession, err
	}
	created, ok := cookieSession.Values["created"].(time.Time)
	if !ok {
		created = time.Now()
	}
	expires, ok := cookieSession.Values["expires"].(time.Time)
	if !ok {
		expires = time.Now().Add(time.Second * time.Duration(s.Opts.MaxAge))
	}
	userID, ok := cookieSession.Values["user_id"].(int)
	if !ok {
		userID = -1
	}

	return PersistedSession{
		Key:     cookieSession.ID,
		UserID:  userID,
		Data:    encoded,
		Created: created,
		Expires: expires,
		Updated: time.Now(),
	}, nil
}

func (s *Store) init() {
	s.initOnce.Do(func() {
		_, err := s.Conn.Exec(
			context.Background(),
			`create table if not exists sessions (key text primary key, user_id int, data text, created timestamp with time zone, updated timestamp with time zone, expires timestamp with time zone)`,
		)
		if err != nil {
			log.Fatalf("[error] failed to create sessions table: %v", err)
		}
	})
}

type PersistedSession struct {
	Key     string
	UserID  int
	Data    string
	Created time.Time
	Updated time.Time
	Expires time.Time
}
