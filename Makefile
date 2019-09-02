.PHONY:
install-tools:
	go get github.com/cespare/reflex

.PHONY: run-dev
run-dev:
	 reflex -s -r '(\.html$$|\.go$$)' -- go run main.go -migrations="file://migrations" -connect="postgres://postgres@localhost:5432/writegood" -verify_key=verify.pem -sign_key=sign.pem -hash_salt=some_salt

.PHONY: migrate-down
migrate-down:
	migrate -source file://migrations -database postgres://postgres@localhost:5432/writegood down 1

.PHONY: migrate-up
migrate-up:
	migrate -source file://migrations -database postgres://postgres@localhost:5432/writegood up 1
