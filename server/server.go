package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/graphql-go/graphql"
	"github.com/jackc/pgx/v4"
)

type User struct {
	ID        int
	Email     string
	Documents []Document
}

type Document struct {
	ID       int
	Text     string
	AuthorID int
}

type Server struct {
	Connect    string
	Migrations string

	conn     *pgx.Conn
	router   *mux.Router
	shutdown chan struct{}
}

// Run the Server.
func (s *Server) Run() error {
	ctx := context.Background()
	var err error
	s.conn, err = pgx.Connect(ctx, s.Connect)

	if err != nil {
		log.Fatalf("[error] failed to connect to database: %v", err)
	}
	defer s.conn.Close(ctx)

	var documentType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Document",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.Int,
				},
				"text": &graphql.Field{
					Type: graphql.String,
				},
			},
		},
	)

	var userType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "User",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.Int,
				},
				"email": &graphql.Field{
					Type: graphql.String,
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
		},
	)

	var schema, _ = graphql.NewSchema(
		graphql.SchemaConfig{
			Query:    queryType,
			Mutation: mutationType,
		},
	)

	s.router = mux.NewRouter()
	s.router.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		log.Printf("[debug] graphql: %s", query)
		result := s.ExecuteQuery(query, schema)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("[error] failed to encode json: %v", err)
		}
	})
	s.router.Handle("/", http.FileServer(http.Dir("./dist")))

	s.shutdown = make(chan struct{}, 1)
	defer func() { <-s.shutdown }()

	log.Printf("running server on :8080")
	return http.ListenAndServe(":8080", s.router)
}

func (s *Server) Shutdown() {
	close(s.shutdown)
}

func (s *Server) MustMigrate() {
	m, err := migrate.New(s.Migrations, s.Connect)
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
