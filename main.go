package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/travisjeffery/writegood/server"
)

func main() {
	var config server.Config

	flag.StringVar(&config.Connect, "connect", "", "db connect string")
	flag.StringVar(&config.Migrations, "migrations", "migrations", "migrations src")
	flag.StringVar(&config.Templates, "templates", "templates", "templates src")
	flag.StringVar(&config.SendGridAPIKey, "sendgrid_api_key", os.Getenv("SENDGRID_API_KEY"), "send grid api key")
	flag.StringVar(&config.Domain, "domain", "http://localhost:8080", "domain")
	flag.StringVar(&config.FromName, "from_name", "Travis Jeffery", "name used to send emails from")
	flag.StringVar(&config.FromAccount, "from_account", "tj@writegood.app", "account used to send emails from")
	flag.DurationVar(&config.SignInExpire, "sign_in_expire", 15*time.Minute, "sign in expire duration")
	flag.StringVar(&config.HashSalt, "hash_salt", "", "hash salt used for sign in tokens")
	flag.StringVar(&config.SignKey, "sign_key", "", "path to sign key")
	flag.StringVar(&config.VerifyKey, "verify_key", "", "path to verify key")

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
