package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	dmp = diffmatchpatch.New()
)

type Store struct {
	Conn *pgx.Conn
}

func (s *Store) FindUserByID(id int) (User, error) {
	log.Printf("[debug] find user with id: %d", id)
	var user User
	err := s.Conn.
		QueryRow(context.Background(), `select id, email, created, updated, signed_in from users where id = $1`, id).
		Scan(&user.ID, &user.Email, &user.Created, &user.Updated, &user.SignedIn)
	return user, err
}

func (s *Store) FindUserByEmail(email string) (User, error) {
	log.Printf("[debug] find user with email: %s", email)
	var user User
	err := s.Conn.
		QueryRow(context.Background(), `select id, email, created, updated, signed_in from users where email = $1`, email).
		Scan(&user.ID, &user.Email, &user.Created, &user.Updated, &user.SignedIn)
	return user, err
}

func (s *Store) CreateUser(email string) (User, error) {
	log.Printf("[debug] create user with email: %s", email)
	var user User
	err := s.Conn.
		QueryRow(context.Background(), `insert into users (email) values ($1) returning id, email, created, updated, signed_in`, email).
		Scan(&user.ID, &user.Email, &user.Created, &user.Updated, &user.SignedIn)
	return user, err
}

func (s *Store) FindLogsByDocumentID(documentID int) ([]DocumentLog, error) {
	log.Printf("[debug] find diffs by document id: %d", documentID)
	var logs []DocumentLog
	rows, err := s.Conn.
		Query(context.Background(), `select id, document_id, text, diffs, diffs_html, type, created, updated from document_logs where document_id = $1`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var log DocumentLog
		if err = rows.Scan(&log.ID, &log.DocumentID, &log.Text, &log.Diffs, &log.DiffsHTML, &log.Type, &log.Created, &log.Updated); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, err
}

func (s *Store) DeleteUser(userID int) error {
	ctx := context.Background()
	_, err := s.Conn.Exec(ctx, `delete from users where id = $1`, userID)
	return err
}

func (s *Store) DeleteDocument(documentID int) error {
	ctx := context.Background()
	_, err := s.Conn.Exec(ctx, `delete from documents where id = $1`, documentID)
	return err
}

func (s *Store) DeleteLogsByDocumentID(documentID int) error {
	ctx := context.Background()
	_, err := s.Conn.Exec(ctx, `delete from document_logs where document_id = $1`, documentID)
	return err
}

func (s *Store) CreateDocument(authorID int, text string) (d Document, err error) {
	log.Printf("[debug] create document with author_id: %d, text: %s", authorID, text)

	ctx := context.Background()

	tx, err := s.Conn.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	err = s.Conn.
		QueryRow(ctx, `insert into documents (text, author_id) values ($1, $2) returning id, text, author_id`, text, authorID).
		Scan(&d.ID, &d.Text, &d.AuthorID)
	if err != nil {
		return
	}

	diffs := d.Diffs(d.Text)
	_, err = s.Conn.
		Exec(ctx,
			`insert into document_logs (document_id, text, diffs_html, diffs, type) values ($1, $2, $3, $4, $5)`,
			d.ID,
			d.Text,
			dmp.DiffPrettyHtml(diffs),
			diffs,
			CreateEvent)
	if err != nil {
		return
	}
	tx.Commit(ctx)

	return d, err
}

func (s *Store) UpdateDocument(id int, text string) (d Document, err error) {
	log.Printf("[debug] update document with id: %d, text: %s", id, text)
	ctx := context.Background()

	tx, err := s.Conn.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	var currText string
	err = s.Conn.QueryRow(ctx, `select text from documents where id = $1`, id).Scan(&currText)
	if err != nil {
		return
	}

	err = s.Conn.
		QueryRow(context.Background(), `update documents set text = $1 where id = $2 returning id, text, author_id`, text, id).
		Scan(&d.ID, &d.Text, &d.AuthorID)
	if err != nil {
		return
	}

	return d, err
}

func (s *Store) UpdateUserSignedIn(id int, signedIn time.Time) (time.Time, error) {
	log.Printf("[debug] update user with id: %d, signed in: %s", id, signedIn)
	err := s.Conn.
		QueryRow(context.Background(), `update users set signed_in = $1, updated = $2 where id = $3 returning signed_in`, signedIn, time.Now(), id).
		Scan(&signedIn)
	return signedIn, err
}

func (s *Store) FindDocumentsByAuthor(authorID int) ([]Document, error) {
	log.Printf("[debug] find documents for author with id: %d", authorID)
	var documents []Document
	rows, err := s.Conn.Query(context.Background(), `select id, text, author_id from documents where author_id = $1`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d Document
		if err = rows.Scan(&d.ID, &d.Text, &d.AuthorID); err != nil {
			return nil, err
		}
		documents = append(documents, d)
	}
	return documents, nil
}

func (s *Store) FindDocumentByID(id int) (Document, error) {
	log.Printf("[debug] find document with id: %d", id)
	var document Document
	err := s.Conn.
		QueryRow(context.Background(), `select id, text, author_id from documents where id = $1`, id).
		Scan(&document.ID, &document.Text, &document.AuthorID)
	return document, err
}

type User struct {
	ID       int        `json:"id"`
	Email    string     `json:"email"`
	SignedIn *time.Time `json:"signed_in"`
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
}

type Document struct {
	ID       int       `json:"id"`
	Text     string    `json:"text"`
	AuthorID int       `json:"author_id"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

func (d Document) Diffs(new string) Diffs {
	return Diffs(dmp.DiffMain(d.Text, new, false))
}

type EventType string

const (
	CreateEvent = "create"
	UpdateEvent = "update"
	ForkEvent   = "fork"
	MergeEvent  = "merge"
)

type Diffs []diffmatchpatch.Diff

func (s *Diffs) Scan(val interface{}) error {
	return scan(s, val)
}

func (s Diffs) Value() (driver.Value, error) {
	return json.Marshal(s)
}

type DocumentLog struct {
	ID         int    `json:"id"`
	DocumentID int    `json:"document_id"`
	Text       string `json:"text"`
	Diffs      Diffs  `json:"diffs"`
	DiffsHTML  string `json:"diffs_html"`
	// Types: create, update, fork, merge
	Type    string    `json:"type"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

func scan(scanner sql.Scanner, val interface{}) error {
	if val == nil {
		return nil
	}
	valRaw := val.([]byte)
	valCopy := make([]byte, len(valRaw))
	copy(valCopy, valRaw)
	return json.Unmarshal(valCopy, scanner)
}
