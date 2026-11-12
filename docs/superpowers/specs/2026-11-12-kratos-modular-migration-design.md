# Migration design: kratos-layout → kratos-modular

Nov 12, 2026

## Intent

Turn this repository, which is currently the stock `kratos-layout` template
(commit `19c645a chore: init project`: one `Todo` resource, SQLite, module
path `kratos-modular`, Go 1.25.7), into the modular-monolith boilerplate
described in
[`2026-11-11-kratos-modular-boilerplate-spec.md`](2026-11-11-kratos-modular-boilerplate-spec.md)
(the "boilerplate spec").

The boilerplate spec defines **what** the end state is. This document defines
**how we get there**: the git history must tell the story of the conversion,
one readable step per commit, so someone reading `git log` months later can
follow how kratos-layout became kratos-modular.

## Goals

- End state matches the boilerplate spec exactly and passes every item of its
  §16 acceptance criteria, each one actually run and reported.
- History on top of `19c645a` is a sequence of small Conventional Commits, each
  one a real step of the conversion (see "Commit sequence").
- Every commit compiles and passes its own static and Docker-free checks, so
  any commit can be checked out, built and reviewed on its own.

## Non-goals

- Anything listed in boilerplate spec §17 (out of scope).
- Changing any decision in the boilerplate spec. Deviate only for a hard
  technical reason, and record the reason in the commit message.

## Anti-goals (failure even if it "works")

- One giant commit that drops in the reference copy.
- Fake evolution: converting `Todo` into a temporary module just to delete it
  later. `Todo` is removed outright in an early commit.
- Commits that do not compile.
- Hand-written code noticeably larger than the reference sizes in boilerplate
  spec §3 (signals scope creep).

## Constraints

- **Reference implementation**: `/Users/namnguyen/Downloads/kratos-modular`.
  It contains the full hand-written code, generated code (`*.pb.go`, Ent,
  `wire_gen.go`, OpenAPI), `migrations/main/*.sql` + `atlas.sum`, and a
  `go.mod` on Go 1.27.4. It has **no `go.sum`** and is not a git repo. Use it as
  the answer key: copy from it per step where it matches the spec; where it and
  the spec disagree, the spec wins. Read-only — never modify it.
- **Environment**: Go 1.27.4 is the active toolchain; Docker 29.8.0 is running.
  `buf`, `wire`, `golangci-lint` are installed.
- **Branching**: all work happens on a new branch `feat/kratos-modular` in its
  own git worktree; `main` is untouched until the user approves the whole
  sequence. Follow the Workflow rules in `AGENTS.md` (subagents work only in
  their task's worktree).
- **`docs/` exception**: boilerplate spec §3 and §17 forbid a `docs/`
  directory. The user explicitly accepted `docs/superpowers/` for these process
  documents (this design, the boilerplate spec, and the implementation plan).
  No other docs are added.
- **`CLAUDE.md`** is a symlink to `AGENTS.md`; edit only `AGENTS.md`.
- **Base commit**: before branching, `main` gets two prep commits on top of
  `19c645a` — `chore: adopt superpowers workflow` (AGENTS.md Workflow section,
  CLAUDE.md symlink) and `docs: add kratos-modular spec and migration design`.
  `feat/kratos-modular` branches from there.
- **Files outside spec §3 that stay**: `LICENSE` (unchanged, MIT, go-kratos)
  and `.gitattributes`, whose generated-file paths are updated to the new
  layout (e.g. `internal/module/*/internal/data/ent/**`, `api/openapi/openapi.yaml`)
  in the commit that moves them.

## Commit sequence

Agreed with the user. Each commit lists what it introduces; details come from
the boilerplate spec sections named.

1. `chore`: module path → `github.com/nam3886/kratos-modular`, `go 1.27.4`,
   and bump only the dependencies already in use (kratos, grpc, protobuf,
   wire, ent) to their §2 versions. `Todo` still builds.
2. `refactor`: remove the `Todo` resource and SQLite, plus the files that only
   served the old layout (`internal/{biz,data,service}/README.md`, root
   `openapi.yaml`); an empty kratos-layout skeleton remains and builds.
3. `feat(platform)`: `internal/platform/{db,mq}`, `internal/platform/migrate/gen`,
   and the new `internal/conf/conf.proto` + `configs/config.yaml` (§5, §7).
   `platform/migrate.Up`, `platform/testkit`, `migrations/migrations.go` and
   `cmd/migrate` are deferred to commit 6, because `//go:embed */*.sql` does
   not compile until the first `.sql` file exists (§15 trap 13). *(Builder
   call: keeps every commit compiling without a placeholder migration.)*
4. `feat(server)`: `internal/server` — JSON codec, filters, metrics,
   messaging, `Registrar`/`Consumer`; `cmd/server` wires an app with zero
   modules (§8, §11).
5. `chore`: tooling needed before modules can generate migrations — full
   `Makefile`, `.golangci.yml`, `docker-compose.yml`, `deploy/postgres/init.sql`,
   `.gitignore`, buf config (§14).
6. `feat(settings)`: `api/settings/v1`, `internal/module/settings`, Ent schema,
   `<timestamp>_settings_init.sql`, its unit tests (§6, §9, §10.1, §12, §13);
   plus the deferred `migrations/migrations.go`, `platform/migrate.Up`,
   `platform/testkit` and `cmd/migrate`.
7. `feat(payment)`: `api/payment/v1`, `internal/module/payment` with consumer,
   `<timestamp>_payment_init.sql`, its unit tests (§10.2).
8. `test`: `cmd/server/integration_test.go` (§13).
9. `docs`: `README.md`, `Dockerfile`, `cmd/migrate` packaging notes, and
   `AGENTS.md` rewritten for the module layout: the five module rules of spec
   §4 as the backbone; the DTO/DO/PO contract kept but restated per module;
   AIP filtering/ordering/pagination and `fieldmask` guidance dropped; the
   Workflow section kept with its gate changed `make all` → `make generate`.

Generated files go in the same commit as their source (AGENTS.md rule).

## Per-commit gate

Every commit, before it is made:

- `go build ./...` and `go vet ./...` pass.
- From commit 5 on: `make lint` reports 0 issues.
- Tests that need no Docker for the code added in that commit pass
  (`biz`, `service`, `server` unit tests).
- Generated code is in sync with its source (re-running the relevant generator
  produces no diff).
- `go mod tidy` leaves `go.mod`/`go.sum` unchanged: each commit adds exactly
  the dependencies its code imports. *(Builder call.)*

Docker-backed tests (`data` layer, integration) and the full §16 list run once
at the end, on the final commit.

## Final acceptance

Run every item of boilerplate spec §16 (A–I), including the "try then revert"
items in B and G, and the items the reference never verified (§15 tail:
`go mod tidy` / `go.sum`, testcontainers branch, docker-compose, Dockerfile,
`make api` / `make config` via buf). Report each item as pass / fail / could
not run, with the reason; never assume.

## Left to the builder

- Exact split of tasks inside each commit, and timestamps of migration files
  (generated by the real `make migration` flow, not copied).
- How to reach each intermediate state from the reference copy, as long as
  each commit honours the gate.
