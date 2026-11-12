# kratos-layout → kratos-modular Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert this kratos-layout repo into the kratos-modular boilerplate through nine commits that each tell one step of the conversion.

**Architecture:** Modular monolith on Kratos v3: shared `internal/platform` (db, mq, migrate, testkit) and `internal/server`; modules `settings` and `payment` under `internal/module/<m>` with their own Wire injectors; payment calls settings in-process, settings notifies payment via outbox → RabbitMQ → inbox.

**Tech Stack:** Go 1.27.4, Kratos v3, Wire, Ent + Atlas + goose, pgx, Watermill (sql/amqp), PostgreSQL 17, RabbitMQ 4, testcontainers, buf, golangci-lint v2.

**Spec:** `docs/superpowers/specs/2026-11-12-kratos-modular-migration-design.md` (how) and `docs/superpowers/specs/2026-11-11-kratos-modular-boilerplate-spec.md` (what; cited below as §N).

## Global Constraints

- Reference implementation (answer key, read-only): `/Users/namnguyen/Downloads/kratos-modular` — written `$REF` below. Copy from it where it matches the spec; where they disagree the spec wins. Never modify `$REF`.
- Module path `github.com/nam3886/kratos-modular`; `go 1.27.4` in `go.mod`; direct dependency versions exactly as §2 (match `$REF/go.mod`).
- Work on branch `feat/kratos-modular` in its own worktree, branched from `main` after the plan commit (`docs: add kratos-modular migration plan`). Never touch the main worktree or its index.
- One task = one commit, Conventional Commits, message ending with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Generated files go in the same commit as their source. Never hand-edit `*.pb.go`, `wire_gen.go`, Ent output, `openapi.yaml`, `atlas.sum`.
- **Per-commit gate** (run before every commit; all must pass):
  `go build ./... && go vet ./... && go mod tidy && git diff --exit-code go.mod go.sum`, plus the task's own Docker-free tests, plus `make lint` from Task 5 on, plus "regenerating the task's generated code gives no diff".
- Docker-backed tests (`data` layer, `integration`) must compile in every commit (`go vet -tags integration ./...`) but are only required to pass in Task 10. Docker is running, so running them earlier is allowed.
- Each commit adds only the dependencies its code imports (tidy stays clean).
- No file or directory outside §3, except `LICENSE` (unchanged), `.gitattributes`, `AGENTS.md`, `CLAUDE.md` (symlink → `AGENTS.md`; edit only `AGENTS.md`), `docs/superpowers/`.
- Hand-written size stays near §3: platform ~560 lines, server ~270, cmd ~190, settings ~260, payment ~400.
- Read §15 (known traps) before Task 3.

## Review Focus

Inputs the spec implies but its test list (§13) does not pin; each has a test added to its owning task:

1. A config naming a database that is not configured (`dbs.Get("nope")`) must return an error, not a nil `*Database` → Task 3, `TestGetUnknownDatabase`.
2. Running migrations twice must be a no-op (outbox/inbox `CREATE ... IF NOT EXISTS`, goose applies 0) → Task 6, `TestUpIsRepeatable`.
3. A GET request has an empty body; the codec must leave the message untouched, not error → Task 4, `TestJSONCodecIgnoresEmptyBody`.
4. `max_body_bytes` negative in config must fall back to 1 MiB, not reject everything → Task 4, extra case in `TestBodyLimitDefaultsToOneMebibyte`.
5. A redelivered `SettingChanged` (same message UUID) arriving through the real consumer path must not add a second `limit_changes` row → covered at the repo by `TestRecordIsIdempotentPerMessage` (Task 7) and end-to-end by §16 E (Task 10).

---

### Task 0: Worktree

- [ ] **Step 1:** From the main checkout: `git worktree add ../kratos-modular-wt -b feat/kratos-modular main`. All later tasks run inside `../kratos-modular-wt`.
- [ ] **Step 2:** Verify `git -C ../kratos-modular-wt log --oneline -1` prints `docs: add kratos-modular migration plan` and `readlink ../kratos-modular-wt/CLAUDE.md` prints `AGENTS.md`.

No commit.

### Task 1 (commit 1): module path, Go 1.27.4, bump existing deps

**Files:**
- Modify: `go.mod`, `go.sum`, `internal/conf/conf.proto` (`go_package` → `github.com/nam3886/kratos-modular/internal/conf;conf`), every `.go`/`.proto` importing `kratos-modular/...`.
- Regenerate: `api/todo/v1/*.pb.go`, `internal/conf/conf.pb.go`, `internal/data/ent/**`, `cmd/kratos-modular/wire_gen.go`, `openapi.yaml`.

- [ ] **Step 1: Baseline** — `go test ./...` currently panics in `api/todo/v1/error_reason.pb.go` init (`slice bounds out of range`). Record the output; this commit must end with it green.
- [ ] **Step 2:** `go mod edit -module github.com/nam3886/kratos-modular -go=1.27.4`; replace import prefix `"kratos-modular/` → `"github.com/nam3886/kratos-modular/` in all `.go` files and `go_package` options in `.proto` files.
- [ ] **Step 3:** Bump only already-required direct deps to §2 versions: `github.com/go-kratos/kratos/v3 v3.0.0`, `github.com/go-kratos/kratos/contrib/otel/v3 v3.0.0-20261026152050-dd0fb7df1e32`, `github.com/google/wire v0.7.0`, `entgo.io/ent v0.14.6`, `google.golang.org/grpc v1.85.0`, `google.golang.org/protobuf v1.36.12`, `go.einride.tech/aip v0.86.3`. Then `go mod tidy`.
- [ ] **Step 4:** Regenerate everything: `make api && make config && go generate ./... && go mod tidy`.
- [ ] **Step 5: Verify** — gate passes and `go test ./...` passes (Todo tests use SQLite, no Docker). If the init panic remains, use superpowers:systematic-debugging; the likely cause is generated code out of sync with the protobuf runtime.
- [ ] **Step 6: Commit** — `chore: move to github.com/nam3886/kratos-modular on Go 1.27.4`.

### Task 2 (commit 2): drop Todo, empty kratos-layout skeleton

**Files:**
- Delete (plain `rm`, then `git add -A` in the worktree): `api/todo/`, `internal/biz/`, `internal/data/`, `internal/service/`, `openapi.yaml`.
- Move: `cmd/kratos-modular/` → `cmd/server/`.
- Modify: `cmd/server/main.go`, `cmd/server/wire.go`, regenerate `cmd/server/wire_gen.go`; `internal/server/{server.go,http.go,grpc.go}`; `.gitattributes`.

**Interfaces:**
- Produces: `server.ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer)`; `NewHTTPServer(*conf.Server) *http.Server`; `NewGRPCServer(*conf.Server) *grpc.Server`; `wireApp(*conf.Server, *conf.Data, *slog.Logger) (*kratos.App, func(), error)`.

- [ ] **Step 1:** Remove the files above; strip Todo registration and the data/biz/service sets from `internal/server` and `cmd/server/wire.go`.
- [ ] **Step 2:** In `cmd/server/main.go` apply §11: remove `env.NewSource("KRATOS")` (keep only `file.NewSource(flagconf)`), remove the `go.uber.org/automaxprocs` import, `defer func() { _ = c.Close() }()` for the config. Keep the `-conf` flag default `../../configs` and the log setup with `tracing.TraceAttrs`.
- [ ] **Step 3:** `.gitattributes`: drop `internal/data/ent/**`; change `openapi.yaml` line to `api/openapi/openapi.yaml`; add `internal/module/*/internal/data/ent/** linguist-generated=true`.
- [ ] **Step 4:** `go run -mod=mod github.com/google/wire/cmd/wire ./cmd/...`, then `go mod tidy` (removes sqlite, mysql, aip-go/ents, automaxprocs).
- [ ] **Step 5: Verify** — gate passes; `go run ./cmd/server -conf ./configs` starts and listens on 8000/9000 (stop it with Ctrl-C).
- [ ] **Step 6: Commit** — `refactor: remove the Todo sample and SQLite`.

### Task 3 (commit 3): platform db, mq, migration generator, new config

Copy from `$REF` and check against §5, §7.1, §7.2, §7.3 (gen only), §12 (`gen.go`).

**Files:**
- Modify: `internal/conf/conf.proto` (exactly §5), regenerate `conf.pb.go`; `configs/config.yaml` (exactly §5).
- Create: `internal/platform/db/db.go`, `internal/platform/db/db_test.go`, `internal/platform/mq/{outbox.go,inbox.go,mq.go}`, `internal/platform/migrate/gen/gen.go`.
- Not yet: `platform/migrate/migrate.go`, `platform/testkit`, `migrations/`, `cmd/migrate` (they need a `.sql` file — §15 trap 13; they land in Task 6).

**Interfaces (Produces):**
- `db.Open(c *conf.Data) (*db.Manager, func(), error)`; `(*Manager).Get(name string) (*db.Database, error)`; `(*Manager).All() []*db.Database`; `db.Database{Name string; Pool *sql.DB; Debug bool}`.
- `mq.Execer` interface; `mq.Publish(ctx, tx mq.Execer, id string, event proto.Message) error`; `mq.Topic(proto.Message) string`; `mq.InitTables(ctx, *sql.DB) error`; `mq.FirstTime(ctx, tx mq.Execer, consumer, messageID string) (bool, error)`; `mq.PoisonTopic = "poison"`; `mq.Handler`; `mq.New(c *conf.Data, dbs *db.Manager, log *slog.Logger) (*mq.MQ, func(), error)`; `(*MQ).Handle(consumer, topic string, h mq.Handler) error`; `(*MQ).Start/Stop(ctx) error`.
- `gen.Main(tables []*schema.Table)`.

- [ ] **Step 1: Failing test** `internal/platform/db/db_test.go`:

```go
func TestGetUnknownDatabase(t *testing.T) {
	dbs, cleanup, err := db.Open(&conf.Data{})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := dbs.Get("nope"); err == nil {
		t.Fatal("Get of an unconfigured database returned no error")
	}
}
```

- [ ] **Step 2:** `go test ./internal/platform/db/` → FAIL (package missing).
- [ ] **Step 3:** Replace `conf.proto` and `config.yaml` per §5; `make config`. Add `db`, `mq`, `migrate/gen` from `$REF`, checking every "Hành vi bắt buộc" bullet of §7.2 (AMQP `ConfirmDelivery = true`, middleware order `PoisonQueue → Retry{3, 100ms, ×2} → Recoverer`, one forwarder per database, `New` creates no tables, `Handle` calls `SubscribeInitialize`).
- [ ] **Step 4:** `go test ./internal/platform/db/` → PASS; gate passes (adds pgx, watermill×3, atlas, lib/pq, uuid as imported).
- [ ] **Step 5: Commit** — `feat(platform): add database, messaging and migration generator`.

### Task 4 (commit 4): server transport, app with zero modules

Copy from `$REF` and check against §8, §11.

**Files:**
- Create/replace: `internal/server/{server.go,http.go,grpc.go,codec.go,codec_test.go,filter.go,filter_test.go,metrics.go,messaging.go}`; `cmd/server/modules.go`.
- Modify: `cmd/server/main.go` (`newApp(logger *slog.Logger, gs *grpc.Server, hs *http.Server, ms *server.Messaging) *kratos.App`, servers `gs, hs, ms`), `cmd/server/wire.go`, regenerate `wire_gen.go`.

**Interfaces:**
- Consumes: Task 3 `db.Open`, `mq.New`, `*mq.MQ`.
- Produces: `server.Registrar{RegisterHTTP(*http.Server); RegisterGRPC(*grpc.Server)}`; `server.Consumer{RegisterConsumers(*mq.MQ) error}`; `server.Messaging{*mq.MQ}`; `NewMessaging(*mq.MQ, []Consumer) (*Messaging, error)`; `NewMetrics() (*Metrics, error)`; `NewHTTPServer(*conf.Server, *Metrics, []Registrar) *http.Server`; `NewGRPCServer(*conf.Server, *Metrics, []Registrar) *grpc.Server`; `ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewMessaging, NewMetrics)`.
- `cmd/server/modules.go` at this commit: `registrars() []server.Registrar` and `consumers() []server.Consumer`, both returning `nil` (no modules yet); Tasks 6–7 add parameters.

- [ ] **Step 1: Failing tests** — copy `codec_test.go` and `filter_test.go` from `$REF` (they implement §13's `TestJSONCodecFollowsTheProtobufJSONMapping`, `TestJSONCodecDecodesIntoABodyField`, `TestBodyLimit`, `TestBodyLimitDefaultsToOneMebibyte`). `$REF`'s codec tests use `paymentv1` messages, which don't exist yet. Until Task 7, make the same assertions with `conf` messages: marshal `&conf.Server_HTTP{Addr: "a", MaxBodyBytes: 500}` must contain `"maxBodyBytes":"500"` and `"network":""`; decode `{"http":{"maxBodyBytes":"7"},"extra":true}` into `&conf.Server{}` gives 7; decode `{"maxBodyBytes":"7","addr":"a"}` into `&srv.Http` (nil field) gives 7 and `a`. Task 7 switches them back to `paymentv1` exactly as §13 states. Add:

```go
func TestJSONCodecIgnoresEmptyBody(t *testing.T) {
	m := &conf.Server_HTTP{Addr: "keep"}
	if err := encoding.GetCodec("json").Unmarshal(nil, m); err != nil {
		t.Fatal(err)
	}
	if m.GetAddr() != "keep" {
		t.Fatalf("empty body changed the message: %v", m)
	}
}
```

and in `TestBodyLimitDefaultsToOneMebibyte` assert `bodyLimit(-1)` also rejects a declared `1<<20 + 1` body with 413 and accepts `1<<20`.
- [ ] **Step 2:** `go test ./internal/server/` → FAIL.
- [ ] **Step 3:** Implement from `$REF`; `codec.go` is exact copy text in §8. Wire `cmd/server` per §11 minus the module providers. Regenerate Wire.
- [ ] **Step 4:** `go test ./internal/server/` → PASS; gate passes.
- [ ] **Step 5: Commit** — `feat(server): add protobuf JSON codec, filters, metrics and messaging`.

### Task 5 (commit 5): tooling

**Files:** replace `Makefile` (§14 table; = `$REF/Makefile`), create `.golangci.yml` (exact §14), `docker-compose.yml` (exact §14), `deploy/postgres/init.sql` (`CREATE DATABASE app_dev;`), replace `.gitignore` (§14 list), `buf.gen.yaml` (openapi `out: api/openapi`).

- [ ] **Step 1:** Write the files.
- [ ] **Step 2: Verify** — `make` prints the help table with every §14 target; `make migration` with no args exits 2 and prints usage; `docker compose config -q` exits 0; `make lint` reports 0 issues (fix any finding in code from Tasks 3–4 inside this commit, per §15 trap 22); gate passes.
- [ ] **Step 3: Commit** — `chore: add Makefile targets, linters and local services`.

### Task 6 (commit 6): settings module + migration runtime

Copy from `$REF` and check against §6 (settings protos), §9, §10.1, §12, §7.3, §7.4, §11 (`cmd/migrate`).

**Files:**
- Create: `api/settings/v1/{settings.proto,events.proto}` + generated; `api/openapi/openapi.yaml` (generated).
- Create: `internal/module/settings/{module.go,wire.go,wire_gen.go}`, `internal/module/settings/internal/service/{service.go,setting.go}`, `.../biz/{biz.go,setting.go,setting_test.go}`, `.../data/{data.go,setting.go,setting_test.go}`, `.../data/ent/{generate.go,schema/setting.go,migrate/main.go}` + Ent output.
- Create: `migrations/main/<timestamp>_settings_init.sql`, `migrations/main/atlas.sum` (both generated), `migrations/migrations.go`, `internal/platform/migrate/{migrate.go,migrate_test.go}`, `internal/platform/testkit/{postgres.go,rabbitmq.go}`, `cmd/migrate/main.go`.
- Modify: `cmd/server/{wire.go,modules.go}`, regenerate `wire_gen.go`.

**Interfaces:**
- Produces: `settings.New(*db.Manager) (*settings.Module, error)`; `(*Module).API() settingsv1.SettingsServiceServer`; `migrate.Up(ctx, *db.Manager, *slog.Logger) error`; `testkit.PostgresURL(testing.TB) string`, `testkit.Postgres(testing.TB) *db.Manager`, `testkit.ResetPostgres(testing.TB, *db.Manager)`, `testkit.Count(testing.TB, *db.Manager, table string) int`, `testkit.RabbitMQURL(testing.TB) string`; `cmd/server/modules.go`: `settingsAPI(*settings.Module) settingsv1.SettingsServiceServer`, `registrars(s *settings.Module) []server.Registrar`.

- [ ] **Step 1:** Protos per §6; `make api`.
- [ ] **Step 2: Failing biz tests** — copy `setting_test.go` (biz) from `$REF`: `TestUpdateThenGet`, `TestUpdateRejectsInvalidSettings` (values in §13). `go test ./internal/module/settings/internal/biz/` → FAIL.
- [ ] **Step 3:** biz, Ent schema (`field.String("id").StorageKey("currency")`, `field.Int64("max_amount")`), `generate.go` with `--feature sql/execquery`, `make ent`; data (`Find`, `Save` exactly as §10.1), service, `module.go`, `wire.go`; `make wire`. Biz tests → PASS.
- [ ] **Step 4: Migration** — `make up`, then `make migration module=settings name=settings_init`. Expect exactly one new `migrations/main/<ts>_settings_init.sql` matching §12's sample and an `atlas.sum`. Do not copy `$REF`'s files.
- [ ] **Step 5:** Add `migrations/migrations.go`, `platform/migrate/migrate.go`, `testkit`, `cmd/migrate/main.go` from `$REF`. Add the data test (`TestSaveStoresSettingAndEventTogether`, §13) and:

```go
func TestUpIsRepeatable(t *testing.T) {
	dbs := testkit.Postgres(t) // already migrated once
	if err := migrate.Up(context.Background(), dbs, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("second Up: %v", err)
	}
	if n := testkit.Count(t, dbs, "settings"); n != 0 {
		t.Fatalf("settings rows = %d, want 0", n)
	}
}
```

in `internal/platform/migrate/migrate_test.go` (package `migrate_test`).
- [ ] **Step 6:** Wire settings into `cmd/server` (`settings.New`, `settingsAPI`, `registrars`), `make wire`.
- [ ] **Step 7: Verify** — gate + `make lint`; `make generate` leaves no diff; `go test ./internal/module/settings/internal/biz/` PASS; `go vet -tags integration ./...` PASS. (Docker tests: `go test ./internal/platform/migrate/ ./internal/module/settings/...` should pass; required in Task 10.)
- [ ] **Step 8: Commit** — `feat(settings): add the settings module and versioned migrations`.

### Task 7 (commit 7): payment module

Copy from `$REF` and check against §6 (payment protos), §10.2.

**Files:**
- Create: `api/payment/v1/{payment.proto,events.proto}` + generated (openapi regenerated).
- Create: `internal/module/payment/{module.go,wire.go,wire_gen.go}`, `.../service/{service.go,payment.go,payment_test.go,setting_events.go}`, `.../biz/{biz.go,payment.go,payment_test.go}`, `.../data/{data.go,payment.go,payment_test.go,limit_change.go,settings_client.go}`, `.../data/ent/{generate.go,schema/{payment.go,limitchange.go},migrate/main.go}` + Ent output.
- Create: `migrations/main/<timestamp>_payment_init.sql` (generated; `atlas.sum` updated).
- Modify: `cmd/server/{wire.go,modules.go}` + `wire_gen.go`; `internal/server/codec_test.go` (switch back to `paymentv1` messages exactly as §13).

**Interfaces:**
- Consumes: `settingsv1.SettingsServiceServer` (Task 6), `mq.Publish`, `mq.FirstTime`, `mq.Topic`, `(*mq.MQ).Handle`.
- Produces: `payment.New(*db.Manager, settingsv1.SettingsServiceServer) (*payment.Module, error)`; `(*Module).RegisterConsumers(*mq.MQ) error` registering consumer `"payment"` on `mq.Topic(&settingsv1.SettingChanged{})`; `cmd/server/modules.go`: `registrars(s *settings.Module, p *payment.Module) []server.Registrar`, `consumers(p *payment.Module) []server.Consumer`.

- [ ] **Step 1:** Protos per §6; `make api`.
- [ ] **Step 2: Failing tests** — copy biz `payment_test.go` (`TestCreate`, table in §13) and service `payment_test.go` (`TestCreateThenGet`, `TestErrorsCarryTheAPIReason`). Run `go test ./internal/module/payment/internal/biz/ ./internal/module/payment/internal/service/` → FAIL.
- [ ] **Step 3:** biz (`Create` rules 1–4 of §10.2 in order; `uuid` is Go 1.27 stdlib in biz, `github.com/google/uuid` only in Ent schema, converted by direct cast in data), Ent schemas, `make ent`, data, service, `module.go`, `wire.go`, `make wire`. Tests → PASS.
- [ ] **Step 4: Migration** — `make migration module=payment name=payment_init`: one file creating `limit_changes` and `payments` only (no statement touching `settings`).
- [ ] **Step 5:** Data tests from `$REF` (`TestCreateStoresPaymentAndEventTogether`, `TestRecordIsIdempotentPerMessage`). Wire payment into `cmd/server` per §11 (`wire.go` list exactly as §11). Switch `codec_test.go` to `paymentv1`.
- [ ] **Step 6: Verify** — gate + `make lint`; `make generate` no diff; Docker-free tests PASS (`go test ./internal/server/ ./internal/module/payment/internal/biz/ ./internal/module/payment/internal/service/ ./internal/module/settings/internal/biz/`); `go vet -tags integration ./...` PASS; `make migrate` applies 2 files, then `make run` starts and `curl -s -X PUT localhost:8000/v1/settings/usd -d '{"currency":"USD","maxAmount":"1000"}'` returns `"maxAmount":"1000"` (stop the server afterwards).
- [ ] **Step 7: Commit** — `feat(payment): add the payment module with limit checks and limit-change events`.

### Task 8 (commit 8): integration test

**Files:** Create `cmd/server/integration_test.go` (`//go:build integration`, package `main`) from `$REF`, checked against §13 `TestPaymentFlow` (ports 18000/19000, the four HTTP calls and expected codes, 15 s wait for `limit_changes` = 1 and `watermill_outbox` = 0, `Run` returns nil after `Stop`).

- [ ] **Step 1:** Add the test.
- [ ] **Step 2: Verify** — `go test -tags integration -run TestPaymentFlow ./cmd/server/` PASS; gate + `make lint`.
- [ ] **Step 3: Commit** — `test: run the whole application against PostgreSQL and RabbitMQ`.

### Task 9 (commit 9): README, Dockerfile, AGENTS.md

**Files:**
- Replace `README.md` per §14 (English, short; sections: intro, Requirements, Quick start with three `curl` examples, Layout, How modules relate (§4's five rules), Adding a module (§11's four steps), Changing the schema (§12), Tests, Configuration, Operations: run `migrate` before server, `poison` exchange needs a bound queue, consumer queues declared at startup, JSON conventions incl. `int64` as string).
- Replace `Dockerfile` per §14 (= `$REF/Dockerfile`).
- Rewrite `AGENTS.md` (not `CLAUDE.md`): project structure = §3 tree (short form); the five module rules of §4 as the backbone; the DTO/DO/PO contract restated per module (service = proto ↔ biz, biz owns entities/errors/repo interfaces and imports only its own `api/<m>/v1`, data = Ent ↔ biz, repo constructors return the biz interface); add-a-module = §11's four steps; schema change = §12's four steps; testing = §13 table; drop AIP filtering/ordering/pagination and `fieldmask`; keep the `## Workflow` section unchanged except the gate: `make all` → `make generate`, and rule b's command stays `go vet` + `go test` per package; keep "Generation & generated files" (`make generate`, `make ent`, `make wire`), Naming, Commits & security.

- [ ] **Step 1:** Write the three files.
- [ ] **Step 2: Verify** — `docker build .` succeeds and `docker run --rm --entrypoint ls <image> /app` lists `server` and `migrate`; every command quoted in README and AGENTS.md exists as a Makefile target or real path (`grep -o 'make [a-z-]*'` against `make help`); `readlink CLAUDE.md` = `AGENTS.md`; gate + `make lint`.
- [ ] **Step 3: Commit** — `docs: document the modular layout for people and agents`.

### Task 10: Final acceptance (no commit unless a fix is needed)

- [ ] **Step 1:** Run every item of §16 A–I on the final commit, including "try then revert" items in B and G (revert with plain file edits; confirm `git status` clean afterwards) and the unverified items in §15's tail. For E use `docker compose stop rabbitmq` / `start`, `kill -9` on the server PID, and `rabbitmqadmin` or the management API on 15672 for the poison and queue checks.
- [ ] **Step 2:** `make test` and `make test-integration` three times in a row, all PASS.
- [ ] **Step 3:** Write the report: each §16 item → PASS / FAIL / NOT RUN with the command and the observed output line that proves it. Any fix goes in a new `fix:` commit on the branch with its own gate.
- [ ] **Step 4:** Whole-branch review (most capable model) against both spec documents, then superpowers:finishing-a-development-branch.
