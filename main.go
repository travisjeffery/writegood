package main

import (
	"flag"
	"log"

	"github.com/travisjeffery/writegood/server"
)

func main() {
	connect := flag.String("connect", "", "db connect string")
	migrations := flag.String("migrations", "", "migrations src")

	flag.Parse()

	log.Printf(`[info] config:
	migrations: %s
	connect: %s
`,
		*migrations,
		*connect)

	s := &server.Server{
		Connect:    *connect,
		Migrations: *migrations,
	}

	s.MustMigrate()

	if err := s.Run(); err != nil {
		log.Fatalf("[error] server failed to run: %v", err)
	}
}
