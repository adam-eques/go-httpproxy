MODULE = $(shell go list -m)
SERVER_NAME = http-proxy-server

.PHONY: generate build test lint build-docker compose compose-down migrate
generate:
	go generate ./...

build: # build a server
	go build -a -o $(SERVER_NAME) $(MODULE)/cmd

test:
	go clean -testcache
	go test ./... -v

run:
	go build -a -o $(SERVER_NAME) $(MODULE)/cmd
	./http-proxy-server

server:
	go run $(MODULE)/cmd
