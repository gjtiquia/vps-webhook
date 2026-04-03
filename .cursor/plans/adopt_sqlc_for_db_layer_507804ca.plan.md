---
name: Adopt sqlc for DB layer
overview: Introduce sqlc for the SQLite database layer in a safe, incremental way by first generating typed query code, then wiring existing handlers/services to use it without changing runtime behavior.
todos:
  - id: add-sql-files
    content: Create schema and query SQL files matching current webhooks behavior
    status: pending
  - id: add-sqlc-config
    content: Add sqlc.yaml configured for SQLite and Go output package
    status: pending
  - id: wire-generation
    content: Document and/or wire sqlc generation command into dev workflow
    status: pending
  - id: refactor-db-wrapper
    content: Refactor internal/db wrapper methods to call generated sqlc queries
    status: pending
  - id: verify-behavior
    content: Run build/tests and verify behavior parity for CRUD and webhook lookup semantics
    status: pending
isProject: false
---

# Adopt sqlc for SQLite access

## Goal
Migrate database access from manual `database/sql` calls to generated, typed query methods via `sqlc`, while keeping current app behavior unchanged.

## Current Baseline
- Current DB logic is hand-written SQL in [`/Users/gjtiquia/Documents/self/vps-webhook/internal/db/db.go`](/Users/gjtiquia/Documents/self/vps-webhook/internal/db/db.go).
- The schema is currently created/altered in Go migration code (`migrate()`), not in standalone SQL migration files.
- Project documentation mentions `sqlc` but it is not configured yet in [`/Users/gjtiquia/Documents/self/vps-webhook/README.md`](/Users/gjtiquia/Documents/self/vps-webhook/README.md).

## Implementation Plan
1. Add SQL source-of-truth files for schema and queries
- Create a schema SQL file (e.g. `db/schema.sql`) that reflects the existing `webhooks` table and `http_method` column defaults.
- Create query SQL files (e.g. `db/query/webhooks.sql`) for all operations currently used by the app: create, list, get-by-path-active, get-by-id, update, and delete.
- Keep SQL semantics aligned with existing behavior (especially defaulting/active-filter logic).

2. Configure sqlc for SQLite + Go generation
- Add `sqlc.yaml` at repo root pointing to schema/query paths and a generated package path (for example `internal/db/sqlc`).
- Set package name and SQL engine config for SQLite so generated APIs match project needs.
- Ensure generated code can be used by existing `*sql.DB` connection setup.

3. Add generation workflow and developer docs
- Add a generation command (e.g. `go generate` directive or documented `sqlc generate` step).
- Update README quick dev workflow with the new generation step and when to run it.
- Keep this lightweight so contributors can regenerate code reliably.

4. Refactor `internal/db` to use generated queries
- Retain current `Open()` and migration entry points initially to avoid runtime changes.
- Replace direct `conn.Exec/Query/QueryRow` calls with `sqlc` generated methods in the DB wrapper.
- Map generated models to existing exported structs (or intentionally switch to generated structs with a small compatibility layer).

5. Validate compatibility and edge cases
- Verify behavior for: default `http_method`, missing webhook path returning nil, active webhook filtering, and update/delete semantics.
- Run existing test/build flow and add focused DB-layer tests if coverage is missing.

## Key Design Decisions
- Prefer incremental migration: keep in-app migration logic first, then optionally move to SQL migration tooling later.
- Keep public DB API shape stable to minimize changes in `cmd/admin` and `cmd/webhook` callers.
- Treat SQL files as canonical for query definitions once sqlc is introduced.

## Files Expected To Change
- [`/Users/gjtiquia/Documents/self/vps-webhook/internal/db/db.go`](/Users/gjtiquia/Documents/self/vps-webhook/internal/db/db.go)
- [`/Users/gjtiquia/Documents/self/vps-webhook/README.md`](/Users/gjtiquia/Documents/self/vps-webhook/README.md)
- New: `/Users/gjtiquia/Documents/self/vps-webhook/sqlc.yaml`
- New: `/Users/gjtiquia/Documents/self/vps-webhook/db/schema.sql`
- New: `/Users/gjtiquia/Documents/self/vps-webhook/db/query/webhooks.sql`
- New generated folder (example): `/Users/gjtiquia/Documents/self/vps-webhook/internal/db/sqlc/`

## Non-Goals (for first pass)
- No large architecture rewrite of handlers/routes.
- No DB engine switch.
- No unrelated UI/admin changes.