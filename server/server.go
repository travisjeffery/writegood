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
	"github.com/travisjeffery/writegood/store"
)

const userSession = "user_session"

type Config struct {
	Connect        string
	Migrations     string
	Templates      string
	VerifyKeyPath  string
	SignKeyPath    string
	SendGridAPIKey string
	Domain         string
	FromAccount    string
	FromName       string
	HashSalt       string
	SignInExpire   time.Duration
	// RawSignKey set in setup.
	RawSignKey []byte
	// SignKey set in setup.
	SignKey *rsa.PrivateKey
	// VerifyKey set in setup.
	VerifyKey *rsa.PublicKey
}

func (c *Config) MustSetup() {
	var err error
	c.RawSignKey, err = ioutil.ReadFile(c.SignKeyPath)
	if err != nil {
		log.Fatalf("[error] failed to read sign key file: %v", err)
	}
	c.SignKey, err = jwt.ParseRSAPrivateKeyFromPEM(c.RawSignKey)
	if err != nil {
		log.Fatalf("[error] failed to parse sign key file: %v", err)
	}
	verifyKey, err := ioutil.ReadFile(c.VerifyKeyPath)
	if err != nil {
		log.Fatalf("[error] failed to read verify key file: %v", err)
	}
	c.VerifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyKey)
	if err != nil {
		log.Fatalf("[error] failed to parse sign key file: %v", err)
	}
}

type Server struct {
	Config Config

	router    *mux.Router
	templates *template.Template
	sessions  sessions.Store
	shutdown  chan struct{}
	email     *sendgrid.Client
	schema    graphql.Schema
	store     *store.Store
}

// Run the Server.
func (s *Server) Run() error {
	ctx := context.Background()
	var err error

	s.Config.MustSetup()

	conn, err := pgx.Connect(ctx, s.Config.Connect)
	if err != nil {
		log.Fatalf("[error] failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	s.store = &store.Store{Conn: conn}

	s.email = sendgrid.NewSendClient(s.Config.SendGridAPIKey)

	sessionStore := sessions.NewCookieStore(s.Config.RawSignKey)
	sessionStore.Options.HttpOnly = true
	sessionStore.Options.Secure = !strings.Contains(s.Config.Domain, "localhost")
	s.sessions = sessionStore

	templateFiles, err := ioutil.ReadDir(s.Config.Templates)
	if err != nil {
		log.Fatalf("[error] failed to read templates dir: %v", err)
	}

	var templateNames []string
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
						return s.store.FindDocumentsByAuthor(p.Source.(store.User).ID)
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
							return s.store.FindUserByID(id)
						}
						email, ok := p.Args["email"].(string)
						if ok {
							return s.store.FindUserByEmail(email)
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
						return s.store.CreateUser(p.Args["email"].(string))
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
						return s.store.CreateDocument(p.Args["author_id"].(int), p.Args["text"].(string))
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
						return s.store.UpdateDocument(p.Args["id"].(int), p.Args["text"].(string))
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
		User *store.User
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
	user, err := s.store.FindUserByEmail(email)
	if err != nil {
		_, _ = io.WriteString(w, "Email not found: "+email)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	now := time.Now()
	signedIn, err := s.store.UpdateUserSignedIn(user.ID, now)
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
	signedToken, err := token.SignedString(s.Config.SignKey)
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

func (s *Server) sessionUser(r *http.Request) *store.User {
	session, err := s.sessions.Get(r, userSession)
	if err != nil {
		return nil
	}
	val, ok := session.Values["user"]
	if !ok {
		return nil
	}
	user := val.(*store.User)
	return user
}

func (s *Server) HandleSignInVerify(w http.ResponseWriter, r *http.Request) {
	// verify sign in
	tokenStr := r.URL.Query().Get("token")
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return s.Config.VerifyKey, nil
	})
	claims := token.Claims.(*Claims)
	user, err := s.store.FindUserByID(claims.UserID)
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

	signedIn, err := s.store.UpdateUserSignedIn(user.ID, time.Now())
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
	gob.Register(&store.User{})
}
