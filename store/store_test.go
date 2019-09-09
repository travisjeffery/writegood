package store_test

import (
	"context"
	"log"
	"testing"

	"github.com/jackc/pgx/v4"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/writegood/store"
)

func TestCreateDocument(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, `postgres://postgres@localhost:5432/writegood`)
	if err != nil {
		log.Fatalf("[error] failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	s := &store.Store{
		Conn: conn,
	}

	uuid := uuid.NewV4()
	user, err := s.CreateUser(uuid.String() + "@example.com")
	require.NoError(t, err)

	defer s.DeleteUser(user.ID)

	document, err := s.CreateDocument(user.ID, "hello world")
	require.NoError(t, err)
	require.Equal(t, "hello world", document.Text)
	require.Equal(t, user.ID, document.AuthorID)

	defer s.DeleteDocument(document.ID)
	defer s.DeleteLogsByDocumentID(document.ID)

	logs, err := s.FindLogsByDocumentID(document.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(logs))
	require.Equal(t, store.CreateEvent, logs[0].Type)
	require.Equal(t, "hello world", logs[0].Text)
	require.Equal(t, "<span>hello world</span>", logs[0].DiffsHTML)
}
