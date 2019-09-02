package session_test

import (
	"context"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/writegood/server/session"
)

func TestStore(t *testing.T) {
	connect := flag.String("connect", "postgres://postgres@localhost:5432/writegood", "db connect string")
	flag.Parse()

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, *connect)
	require.NoError(t, err)
	defer conn.Close(ctx)
	_, _ = conn.Exec(context.Background(), `delete from sessions`)

	store := &session.Store{
		Conn:   conn,
		Codecs: securecookie.CodecsFromPairs([]byte("hi")),
		Opts: &sessions.Options{
			Secure: false,
			MaxAge: 60 * 60 * 24 * 30,
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	name := "the-session"

	// create new cookie
	session, err := store.Get(req, name)
	require.Equal(t, err, http.ErrNoCookie)

	session.Values["user_id"] = 1

	w := httptest.NewRecorder()
	err = store.Save(req, w, session)
	require.NoError(t, err)

	set := strings.Split(w.Header().Get("Set-Cookie"), "=")
	require.Equal(t, name, set[0])

	// get the cookie
	req, _ = http.NewRequest("GET", "http://example.com", nil)
	w = httptest.NewRecorder()
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, store.Codecs...)
	require.NoError(t, err)
	req.AddCookie(sessions.NewCookie(session.Name(), encoded, session.Options))
	session, err = store.Get(req, session.Name())
	require.NoError(t, err)

	// update the cookie
	email := "tj@example.com"
	session.Values["user_email"] = email
	w = httptest.NewRecorder()
	err = store.Save(req, w, session)
	require.NoError(t, err)

	session, err = store.Get(req, session.Name())
	require.NoError(t, err)
	require.Equal(t, email, session.Values["user_email"])
}
