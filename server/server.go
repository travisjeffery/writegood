package server

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/graphql-go/graphql"
	"github.com/jackc/pgx/v4"
	"github.com/sendgrid/sendgrid-go"
)

const userSession = "user_session"

type User struct {
	ID       int        `json:"id"`
	Email    string     `json:"email"`
	SignedIn *time.Time `json:"signed_in"`
	Created  time.Time  `json:"signed_in"`
	Updated  time.Time  `json:"signed_in"`
}

type Document struct {
	ID       int       `json:"id"`
	Text     string    `json:"text"`
	AuthorID int       `json:"author_id"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

type Config struct {
	Connect        string
	Migrations     string
	Templates      string
	VerifyKey      string
	SignKey        string
	SendGridAPIKey string
	Domain         string
	FromAccount    string
	FromName       string
	HashSalt       string
	SignInExpire   time.Duration

	signKey   *rsa.PrivateKey
	verifyKey *rsa.PublicKey
}

type Server struct {
	Config Config

	conn      *pgx.Conn
	router    *mux.Router
	templates *template.Template
	sessions  sessions.Store
	shutdown  chan struct{}
	email     *sendgrid.Client
	schema    graphql.Schema
}

// Run the Server.
func (s *Server) Run() error {
	ctx := context.Background()
	var err error

	s.conn, err = pgx.Connect(ctx, s.Config.Connect)
	if err != nil {
		log.Fatalf("[error] failed to connect to database: %v", err)
	}
	defer s.conn.Close(ctx)

	s.email = sendgrid.NewSendClient(s.Config.SendGridAPIKey)

	signKey, err := ioutil.ReadFile(s.Config.SignKey)
	if err != nil {
		log.Fatalf("[error] failed to read sign key file: %v", err)
	}
	s.Config.signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signKey)
	if err != nil {
		log.Fatalf("[error] failed to parse sign key file: %v", err)
	}
	verifyKey, err := ioutil.ReadFile(s.Config.VerifyKey)
	if err != nil {
		log.Fatalf("[error] failed to read verify key file: %v", err)
	}
	s.Config.verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyKey)
	if err != nil {
		log.Fatalf("[error] failed to parse sign key file: %v", err)
	}

	store := sessions.NewCookieStore(signKey)
	store.Options.HttpOnly = true
	store.Options.Secure = !strings.Contains(s.Config.Domain, "localhost")
	s.sessions = store

	templateFiles, err := ioutil.ReadDir(s.Config.Templates)
	var templateNames []string
	if err != nil {
		log.Fatalf("[error] failed to read templates dir: %v", err)
	}
	for _, f := range templateFiles {
		templateNames = append(templateNames, path.Join(s.Config.Templates, f.Name()))
	}
	s.templates, err = template.ParseFiles(templateNames...)
	if err != nil {
		log.Fatalf("[error] failed to parse templates: %v", err)
	}

	var documentType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Document",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.NewNonNull(graphql.Int),
				},
				"text": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"created": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"updated": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},
				"author_id": &graphql.Field{
					Type: graphql.NewNonNull(graphql.Int),
				},
			},
		},
	)

	var userType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "User",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.NewNonNull(graphql.Int),
				},
				"email": &graphql.Field{
					Type: graphql.NewNonNull(graphql.String),
				},

				"documents": &graphql.Field{
					Type: graphql.NewList(documentType),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return s.FindDocumentsByAuthor(p.Source.(User).ID)
					},
				},
			},
		},
	)

	var queryType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"user": &graphql.Field{
					Type:        userType,
					Description: "get user",
					Args: graphql.FieldConfigArgument{
						"id": &graphql.ArgumentConfig{
							Type: graphql.Int,
						},
						"email": &graphql.ArgumentConfig{
							Type: graphql.String,
						},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						id, ok := p.Args["id"].(int)
						if ok {
							return s.FindUserByID(id)
						}
						email, ok := p.Args["email"].(string)
						if ok {
							return s.FindUserByEmail(email)
						}
						return nil, fmt.Errorf("neither id nor email arg set")
					},
				},
			},
		},
	)

	var mutationType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Mutation",
			Fields: graphql.Fields{
				"createUser": &graphql.Field{
					Type:        userType,
					Description: "Create a user.",
					Args: graphql.FieldConfigArgument{
						"email": &graphql.ArgumentConfig{
							Type: graphql.NewNonNull(graphql.String),
						},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return s.CreateUser(p.Args["email"].(string))
					},
				},
				"createDocument": &graphql.Field{
					Type:        documentType,
					Description: "Create a document.",
					Args: graphql.FieldConfigArgument{
						"text": &graphql.ArgumentConfig{
							Type: graphql.NewNonNull(graphql.String),
						},
						"author_id": &graphql.ArgumentConfig{
							Type: graphql.NewNonNull(graphql.Int),
						},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return s.CreateDocument(p.Args["author_id"].(int), p.Args["text"].(string))
					},
				},
				"updateDocument": &graphql.Field{
					Type:        documentType,
					Description: "Update a document.",
					Args: graphql.FieldConfigArgument{
						"text": &graphql.ArgumentConfig{
							Type: graphql.NewNonNull(graphql.String),
						},
						"id": &graphql.ArgumentConfig{
							Type: graphql.NewNonNull(graphql.Int),
						},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return s.UpdateDocument(p.Args["id"].(int), p.Args["text"].(string))
					},
				},
			},
		},
	)

	s.schema, err = graphql.NewSchema(
		graphql.SchemaConfig{
			Query:    queryType,
			Mutation: mutationType,
		},
	)
	if err != nil {
		log.Fatalf("[error] failed to create schema: %v", err)
	}

	s.router = mux.NewRouter()

	s.router.PathPrefix("/static").
		Handler(http.StripPrefix("/static", http.FileServer(http.Dir("dist"))))
	s.router.HandleFunc("/graphql", s.HandleGraphql)
	s.router.HandleFunc("/sign_in", s.HandleSignIn).Methods("POST")
	s.router.HandleFunc("/sign_in/verify", s.HandleSignInVerify)
	s.router.HandleFunc("/sign_out", s.HandleSignOut)
	s.router.HandleFunc("/", s.HandleHomepage)

	s.shutdown = make(chan struct{}, 1)
	defer func() { <-s.shutdown }()

	log.Printf("running server on :8080")
	return http.ListenAndServe(":8080", s.router)
}

func (s *Server) Shutdown() {
	close(s.shutdown)
}

func (s *Server) MustMigrate() {
	m, err := migrate.New(s.Config.Migrations, s.Config.Connect)
	if err != nil {
		log.Fatalf("[error] failed to create migrate instance: %v", err)
	}
	err = m.Up()
	if err == migrate.ErrNoChange {
		return
	}
	if err != nil {
		log.Fatalf("[error] failed to migrate: %v", err)
	}
}

type Claims struct {
	UserID int
	Hash   string
	jwt.StandardClaims
}

func (s *Server) generateSignInHash(userID int, signedIn time.Time) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(fmt.Sprintf("%s%d%d", s.Config.HashSalt, userID, signedIn.Nanosecond()))),
	)
}

type jsonQuery struct {
	Query string `json:"query"`
}

func (s *Server) HandleHomepage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		User *User
	}{
		User: s.sessionUser(r),
	}
	spew.Dump(data)
	if err := s.templates.Lookup("homepage.html").Execute(w, data); err != nil {
		log.Printf("[error] failed to execute template: %v", err)
	}
}

func (s *Server) HandleGraphql(w http.ResponseWriter, r *http.Request) {
	// session, err := s.sessions.Get(r, userSession)
	// if err != nil {
	// 	log.Printf("[error] failed to get session: %v", err)
	// }
	// user := session.Values["user"].(User)

	query := r.URL.Query().Get("query")
	// TODO: better way to handle this?
	if r.Header.Get("Content-Type") == "application/json" {
		var q jsonQuery
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		query = q.Query
	}

	// log.Printf("[debug] graphql query for user: %d: query: %s", user, query)

	result := s.ExecuteQuery(query, s.schema)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("[error] failed to encode json: %v", err)
	}
}

func (s *Server) HandleSignIn(w http.ResponseWriter, r *http.Request) {
	// send email to log in
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	email := r.Form.Get("email")
	user, err := s.FindUserByEmail(email)
	if err != nil {
		_, _ = io.WriteString(w, "Email not found: "+email)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	now := time.Now()
	signedIn, err := s.UpdateUserSignedIn(user.ID, now)
	if err != nil {
		log.Printf("[error] failed to update user signed in: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hash := s.generateSignInHash(user.ID, signedIn)
	claims := Claims{
		UserID: user.ID,
		Hash:   hash,
		StandardClaims: jwt.StandardClaims{
			Issuer:    "WriteGood",
			Id:        uuid.NewV4().String(),
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(s.Config.SignInExpire).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(s.Config.signKey)
	if err != nil {
		log.Printf("[error] failed to sign token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data := struct {
		Token  string
		Domain string
	}{
		Token:  signedToken,
		Domain: s.Config.Domain,
	}
	var plain, html bytes.Buffer
	if err = s.templates.Lookup("sign_in_plain.html").Execute(&plain, data); err != nil {
		log.Printf("[error] failed to execute plain email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err = s.templates.Lookup("sign_in_html.html").Execute(&html, data); err != nil {
		log.Printf("[error] failed to execute html email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	mail := mail.NewSingleEmail(
		mail.NewEmail(s.Config.FromName, s.Config.FromAccount),
		"Sign in to WriteGood",
		mail.NewEmail("", email),
		plain.String(),
		html.String(),
	)
	_, err = s.email.Send(mail)
	if err != nil {
		log.Printf("[error] failed to send email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("[debug] sent sign in verify email to user: %v", email)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) sessionUser(r *http.Request) *User {
	session, err := s.sessions.Get(r, userSession)
	if err != nil {
		return nil
	}
	val, ok := session.Values["user"]
	if !ok {
		return nil
	}
	user := val.(*User)
	return user
}

func (s *Server) HandleSignInVerify(w http.ResponseWriter, r *http.Request) {
	// verify sign in
	tokenStr := r.URL.Query().Get("token")
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return s.Config.verifyKey, nil
	})
	claims := token.Claims.(*Claims)
	user, err := s.FindUserByID(claims.UserID)
	if err != nil {
		log.Printf("[error] failed to find user by id: %d: %v", claims.UserID, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if claims.Hash != s.generateSignInHash(user.ID, *user.SignedIn) {
		log.Printf("[error] failed to verify claims for user: %d", user.ID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf("[debug] verified sign in of user: %d", user.ID)

	// create session
	session, err := s.sessions.Get(r, userSession)
	if err != nil {
		log.Printf("[error] failed to get session: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session.Values["user"] = &user
	if err = s.sessions.Save(r, w, session); err != nil {
		log.Printf("[error] failed to save session: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf("[debug] saved session: %d", user.ID)

	signedIn, err := s.UpdateUserSignedIn(user.ID, time.Now())
	if err != nil {
		log.Printf("[error] failed to verify claims for user: %d", user.ID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user.SignedIn = &signedIn
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) HandleSignOut(w http.ResponseWriter, r *http.Request) {
	session, err := s.sessions.Get(r, userSession)
	if err != nil {
		log.Printf("[error] failed to get session: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	session.Options.MaxAge = -1
	if err = s.sessions.Save(r, w, session); err != nil {
		log.Printf("[error] failed to save session: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) FindUserByID(id int) (User, error) {
	log.Printf("[debug] find user with id: %d", id)
	var user User
	err := s.conn.
		QueryRow(context.Background(), `select id, email, created, updated, signed_in from users where id = $1`, id).
		Scan(&user.ID, &user.Email, &user.Created, &user.Updated, &user.SignedIn)
	return user, err
}

func (s *Server) FindUserByEmail(email string) (User, error) {
	log.Printf("[debug] find user with email: %s", email)
	var user User
	err := s.conn.
		QueryRow(context.Background(), `select id, email, created, updated, signed_in from users where email = $1`, email).
		Scan(&user.ID, &user.Email, &user.Created, &user.Updated, &user.SignedIn)
	return user, err
}

func (s *Server) CreateUser(email string) (interface{}, error) {
	log.Printf("[debug] create user with email: %s", email)
	var user User
	err := s.conn.
		QueryRow(context.Background(), `insert into users (email) values ($1) returning id, email, created, updated, signed_in`, email).
		Scan(&user.ID, &user.Email, &user.Created, &user.Updated, &user.SignedIn)
	return user, err
}

func (s *Server) CreateDocument(authorID int, text string) (interface{}, error) {
	log.Printf("[debug] create document with author_id: %d, text: %s", authorID, text)
	var d Document
	err := s.conn.
		QueryRow(context.Background(), `insert into documents (text, author_id) values ($1, $2) returning id, text, author_id`, text, authorID).
		Scan(&d.ID, &d.Text, &d.AuthorID)
	return d, err
}

func (s *Server) UpdateDocument(id int, text string) (interface{}, error) {
	log.Printf("[debug] update document with id: %d, text: %s", id, text)
	var d Document
	err := s.conn.
		QueryRow(context.Background(), `update documents set text = $1 where id = $2 returning id, text, author_id`, text, id).
		Scan(&d.ID, &d.Text, &d.AuthorID)
	return d, err
}

func (s *Server) UpdateUserSignedIn(id int, signedIn time.Time) (time.Time, error) {
	log.Printf("[debug] update user with id: %d, signed in: %s", id, signedIn)
	err := s.conn.
		QueryRow(context.Background(), `update users set signed_in = $1, updated = $2 where id = $3 returning signed_in`, signedIn, time.Now(), id).
		Scan(&signedIn)
	return signedIn, err
}

func (s *Server) FindDocumentsByAuthor(authorID int) (interface{}, error) {
	log.Printf("[debug] find documents for author with id: %d", authorID)
	var documents []Document
	rows, err := s.conn.Query(context.Background(), `select id, text, author_id from documents where author_id = $1`, authorID)
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

func (s *Server) FindDocumentByID(id int) (interface{}, error) {
	log.Printf("[debug] find document with id: %d", id)
	var document Document
	err := s.conn.
		QueryRow(context.Background(), `select id, text, author_id from documents where id = $1`, id).
		Scan(&document.ID, &document.Text, &document.AuthorID)
	return document, err
}

func (s *Server) ExecuteQuery(query string, schema graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})
	if len(result.Errors) > 0 {
		log.Printf("[error] errors: %v", result.Errors)
	}
	return result
}

func init() {
	// so we can write users to session values
	gob.Register(&User{})
}
