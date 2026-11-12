VERSION=$(shell git describe --tags --always 2>/dev/null || echo dev)

# Arguments of `make migration`.
module ?=
name ?=
db ?= main

.PHONY: init
# install the code generators
init:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: api
# generate Go code and the OpenAPI document from api/
api:
	buf generate --template buf.gen.yaml

.PHONY: config
# generate Go code from internal/conf/conf.proto
config:
	buf generate --template buf.gen.config.yaml

.PHONY: ent
# generate the ent code of every module
ent:
	go generate ./internal/module/*/internal/data/ent

.PHONY: wire
# generate the wiring of every module and of the application
wire:
	go run -mod=mod github.com/google/wire/cmd/wire ./internal/module/... ./cmd/...
	go mod tidy

.PHONY: generate
# generate everything
generate: api config ent wire
	go mod tidy

.PHONY: build
# build the binaries into ./bin
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./cmd/...

.PHONY: up
# start PostgreSQL and RabbitMQ
up:
	docker compose up -d --wait

.PHONY: down
# stop PostgreSQL and RabbitMQ
down:
	docker compose down

.PHONY: migration
# write a migration from a module's schema changes: make migration module=payment name=add_note
migration:
	@test -n "$(module)" -a -n "$(name)" || (echo "usage: make migration module=<module> name=<what_changed> [db=main]"; exit 2)
	go run -mod=mod ./internal/module/$(module)/internal/data/ent/migrate/main.go -name $(name) -db $(db)

.PHONY: migrate
# apply pending migrations to every configured database
migrate:
	go run ./cmd/migrate -conf ./configs

.PHONY: run
# run the server
run:
	go run ./cmd/server -conf ./configs

.PHONY: test
# run the unit and data tests (needs Docker)
test:
	go test ./...

.PHONY: test-integration
# run all tests, including the one that starts the whole application (needs Docker)
test-integration:
	go test -tags integration ./...

.PHONY: lint
# run the linters
lint:
	golangci-lint run ./...

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
