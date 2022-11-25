MODULE = $(shell go list -m)
SERVER_NAME = http-proxy

.PHONY: generate build test lint build-docker compose compose-down migrate
generate:
	go generate ./...

build: # build a server
	go build -a -o $(SERVER_NAME) $(MODULE)/cmd

test:
	go clean -testcache
	go test ./... -v

server:
	go run $(MODULE)/cmd
