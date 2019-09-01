package main

import (
	"flag"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/travisjeffery/writegood/server"
)

func main() {
	var config server.Config

	flag.StringVar(&config.Connect, "connect", "", "db connect string")
	flag.StringVar(&config.Migrations, "migrations", "", "migrations src")

	flag.Parse()

	log.Printf("[info] config:\n%s", spew.Sdump(config))

	s := &server.Server{
		Config: config,
	}

	s.MustMigrate()

	if err := s.Run(); err != nil {
		log.Fatalf("[error] server failed to run: %v", err)
	}
}
