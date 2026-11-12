# kratos-modular

A modular-monolith starter on [Kratos v3](https://github.com/go-kratos/kratos): one
deployable application made of modules that the compiler keeps apart.

- **API**: proto first. One service definition gives HTTP, gRPC and an OpenAPI document.
- **Modules**: each module is a small Kratos layout (`service`, `biz`, `data`) behind its own `internal/`.
- **Data**: ent on PostgreSQL. Any number of databases, one shared pool each, versioned SQL migrations.
- **Messaging**: transactional outbox, RabbitMQ, consumers with retry, a poison queue and an inbox ([Watermill](https://watermill.io)).
- **Operations**: Prometheus metrics, CORS, request size limit, graceful shutdown.

Two sample modules show both ways modules talk to each other: `payment` calls
`settings` directly to read a limit, and `settings` tells `payment` about
changes with an event.

## Requirements

Go 1.27, Docker (local services and tests), and the tools installed by `make init`.

## Quick start

```sh
go mod tidy       # first time only: resolves and records dependency versions
make init         # install wire, buf, golangci-lint
make up           # PostgreSQL and RabbitMQ
make migrate      # create the tables
make run          # HTTP on :8000, gRPC on :9000
```

```sh
curl -X PUT localhost:8000/v1/settings/usd -H 'Content-Type: application/json' \
  -d '{"maxAmount":"1000"}'
curl -X POST localhost:8000/v1/payments -H 'Content-Type: application/json' \
  -d '{"amount":"500","currency":"usd"}'
curl localhost:8000/metrics
```

`make help` lists every target.

## Use it as a template

1. Rename the module: `go mod edit -module example.com/acme/shop`
2. Rewrite the import paths:
   `git grep -lF github.com/nam3886/kratos-modular -- '*.go' '*.proto' .golangci.yml | xargs perl -pi -e 's#github\.com/nam3886/kratos-modular#example.com/acme/shop#g'`
3. Regenerate the code that embeds the old path: `make generate`

## Layout

```
api/<module>/v1/          proto: service, messages, error reasons, events (+ generated code)
api/openapi/openapi.yaml  generated API document
cmd/server/               the application; modules.go and wire.go list the modules
cmd/migrate/              applies migrations
configs/config.yaml       the only configuration source
migrations/<database>/    versioned SQL, one directory per configured database
internal/conf/            config schema
internal/server/          HTTP and gRPC servers, JSON codec, CORS, body limit, metrics, messaging
internal/platform/db/       opens every configured database once
internal/platform/mq/       outbox, forwarder, consumers, inbox
internal/platform/migrate/  applies and generates migrations
internal/platform/testkit/  PostgreSQL and RabbitMQ for tests
internal/module/<module>/
    module.go             what the rest of the application may use
    wire.go               wires the module's own layers
    internal/service/     implements the API; handles incoming events
    internal/biz/         rules, entities, and the interfaces biz needs
    internal/data/        repositories, ent schema, adapters to other modules
```

## How modules relate

- **Boundary.** Everything under `internal/module/<m>/internal/` is invisible outside
  that module; the compiler enforces it. Other code may import only
  `internal/module/<m>` and `api/<m>/v1`.
- **Direct calls go one way.** The caller declares the interface it needs in its
  `biz` package and implements it in `data` with the other module's generated
  server interface (`payment/internal/data/settings_client.go`). The call is a Go
  call in the same process. If two modules depend on each other, `wire` fails with
  a cycle error.
- **The other direction is an event.** A repository stores the row and publishes
  the event in one transaction (`mq.Publish`). The forwarder delivers it to
  RabbitMQ after the commit. Delivery is at least once, so a handler records the
  message in the inbox inside its own transaction (`mq.FirstTime`) and does
  nothing on a repeat.
- **Transactions stay inside one repository method.** There is no transaction
  across modules.
- **Databases.** A module picks its database with the `database` constant in its
  `data.go`. Modules on the same database share one connection pool.

## Adding a module

1. Write `api/<name>/v1/*.proto`, then `make api`.
2. Copy `internal/module/settings` (or `payment` if it consumes events) to
   `internal/module/<name>` and replace its contents; a module with consumers
   must also rename `data.Consumer` in its `data.go`.
3. Add the module to `cmd/server/wire.go` and `cmd/server/modules.go`, then
   `make ent wire`.
4. `make migration module=<name> name=<name>_init`, then `make migrate`.

## Changing the schema

Edit the ent schema, then:

```sh
make ent
make migration module=payment name=add_note   # writes migrations/main/<timestamp>_add_note.sql
make migrate
```

`make migration` compares the module's schema with the migration directory of
its database, replayed on the empty `app_dev` database. Hand-written migration
files are fine: put them in the same directory using the goose format.

`make migration` also writes dropped columns and indexes. Removing a whole
entity does not drop its table: write that `DROP TABLE` by hand. Declare every
index of an Ent table in its Ent schema; a hand-written index that is missing
there is dropped by the next `make migration`.

The outbox and inbox tables are created by `make migrate` as well.

## Tests

| Layer | How |
|---|---|
| `biz`, `service` | plain Go with hand-written fakes |
| `data` | real PostgreSQL, migrated with the real migrations |
| `cmd/server` (tag `integration`) | the whole application with real PostgreSQL and RabbitMQ |

`make test` and `make test-integration` start the services with testcontainers.

## Configuration

`configs/config.yaml` is the only source. Notable keys:

| Key | Meaning |
|---|---|
| `server.http.cors_allowed_origins` | origins allowed to call the API from a browser; empty disables CORS |
| `server.http.max_body_bytes` | largest request body; default 1 MiB |
| `data.databases.<name>` | one entry per database: `driver`, `source`, `max_open_conns`, `debug` |
| `data.rabbitmq.url` | RabbitMQ connection |

## Operations

- `GET /metrics` serves request counts and durations per operation, plus Go
  runtime and process metrics.
- Run `migrate` once per release, before the new server version starts.
- Messages a consumer cannot handle after three retries go to the `poison`
  exchange, whose durable queue `poison_poison` is declared at startup. Replay
  from it once the cause is fixed. A message interrupted by shutdown is not
  poisoned: RabbitMQ redelivers it after restart.
- A consumer's queue is declared when the application starts. Events published
  before a consumer's first start are not delivered to it.
- JSON follows the protobuf JSON mapping, as the OpenAPI document describes it:
  camelCase names, 64-bit integers as strings, RFC 3339 timestamps.
