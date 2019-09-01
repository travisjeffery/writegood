.PHONY:
install-tools:
	 go get github.com/cespare/reflex

.PHONY: run-dev
run-dev:
	 reflex -s -r '(\.html$|\.go$)' -- go run main.go -migrations="file://migrations" -connect="postgres://postgres@localhost:5432/writegood"
