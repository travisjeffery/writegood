package server

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/graphql-go/graphql"
	"github.com/jackc/pgx/v4"
	"github.com/sendgrid/sendgrid-go"
)

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

type Document struct {
	ID       int    `json:"id"`
	Text     string `json:"text"`
	AuthorID int    `json:"author_id"`
}

type Config struct {
	Connect        string
	Migrations     string
	Templates      string
	SessionKey     string
	SendGridAPIKey string
	Domain         string
}

type Server struct {
	Config Config

	conn      *pgx.Conn
	router    *mux.Router
	templates *template.Template
	sessions  sessions.Store
	shutdown  chan struct{}
	email     *sendgrid.Client
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

	f, err := os.Open(s.Config.SessionKey)
	if err != nil {
		log.Fatalf("[error] failed to open session key file: %v", err)
	}
	sessionKey, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("[error] failed to read session key file: %v", err)
	}
	store := sessions.NewCookieStore(sessionKey)
	store.Options.HttpOnly = true
	store.Options.Secure = true
	s.sessions = store

	templateFiles, err := ioutil.ReadDir(s.Config.Templates)
	var templateNames []string
	if err != nil {
		log.Fatalf("[error] failed to read templates dir: %v", err)
	}
	for _, f := range templateFiles {
		templateNames = append(templateNames, path.Join(s.Config.Templates, f.Name()))
	}
	templates, err := template.ParseFiles(templateNames...)
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
					Type:    graphql.NewList(documentType),
					Resolve: s.FindDocumentsByAuthor,
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
					Description: "get user by id",
					Args: graphql.FieldConfigArgument{
						"id": &graphql.ArgumentConfig{
							Type: graphql.Int,
						},
					},
					Resolve: s.FindUserByID,
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
					Resolve: s.CreateUser,
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
					Resolve: s.CreateDocument,
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
					Resolve: s.UpdateDocument,
				},
			},
		},
	)

	schema, err := graphql.NewSchema(
		graphql.SchemaConfig{
			Query:    queryType,
			Mutation: mutationType,
		},
	)
	if err != nil {
		log.Fatalf("[error] failed to create schema: %v", err)
	}

	type jsonQuery struct {
		Query string `json:"query"`
	}

	s.router = mux.NewRouter()

	s.router.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[debug] graphql: %s", query)
		result := s.ExecuteQuery(query, schema)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("[error] failed to encode json: %v", err)
		}
	})

	fs := http.FileServer(http.Dir("dist"))
	s.router.PathPrefix("/static").Handler(http.StripPrefix("/static", fs))

	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := templates.Lookup("index.html").Execute(w, nil); err != nil {
			log.Printf("[error] failed to execute template: %v", err)
		}
	})

	s.router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		// send email to log in
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		email := r.Form.Get("email")
		fmt.Printf("login with email: %s\n", email)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}).Methods("POST")

	s.router.HandleFunc("/verify_login", func(w http.ResponseWriter, r *http.Request) {
		// send email to log in
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		email := r.Form.Get("email")
		fmt.Printf("login with email: %s\n", email)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}).Methods("GET")

	s.router.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		// delete session
	}).Methods("POST")

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

func (s *Server) FindUserByID(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(int)
	if !ok {
		return nil, fmt.Errorf("id isn't an int")
	}
	log.Printf("[debug] find user with id: %d", id)
	var user User
	err := s.conn.
		QueryRow(context.Background(), `select id, email from users where id = $1`, id).
		Scan(&user.ID, &user.Email)
	return user, err
}

func (s *Server) CreateUser(p graphql.ResolveParams) (interface{}, error) {
	log.Printf("[debug] create user with email: %s", p.Args["email"])
	var u User
	err := s.conn.
		QueryRow(context.Background(), `insert into users (email) values ($1) returning id, email`, p.Args["email"]).
		Scan(&u.ID, &u.Email)
	return u, err
}

func (s *Server) CreateDocument(p graphql.ResolveParams) (interface{}, error) {
	log.Printf("[debug] create document with author_id: %d, text: %s", p.Args["author_id"], p.Args["text"])
	var d Document
	err := s.conn.
		QueryRow(context.Background(), `insert into documents (text, author_id) values ($1, $2) returning id, text, author_id`, p.Args["text"], p.Args["author_id"]).
		Scan(&d.ID, &d.Text, &d.AuthorID)
	return d, err
}

func (s *Server) UpdateDocument(p graphql.ResolveParams) (interface{}, error) {
	log.Printf("[debug] update document with id: %d, text: %s", p.Args["id"], p.Args["text"])
	var d Document
	err := s.conn.
		QueryRow(context.Background(), `update documents set text = $1 where id = $2 returning id, text, author_id`, p.Args["text"], p.Args["id"]).
		Scan(&d.ID, &d.Text, &d.AuthorID)
	return d, err
}

func (s *Server) FindDocumentsByAuthor(p graphql.ResolveParams) (interface{}, error) {
	user := p.Source.(User)
	log.Printf("[debug] find documents for author with id: %d", user.ID)
	var documents []Document
	rows, err := s.conn.Query(context.Background(), `select id, text, author_id from documents where author_id = $1`, user.ID)
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

func (s *Server) FindDocumentByID(p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(int)
	if !ok {
		return nil, fmt.Errorf("id isn't an int")
	}
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
