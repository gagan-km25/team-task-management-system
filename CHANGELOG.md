# Changelog

## 2026-10-29

### Documentation
- Final documentation pass across the root README and all `docs/01`–`docs/16` plus `CLAUDE.md`: pending-registration feature documented everywhere relevant; acceptance statuses aligned to the PASS/FAIL/PARTIAL/BLOCKED/NOT VERIFIED scheme (AC9 and AC14 marked PARTIAL — no automated browser evidence; user-reported manual testing recorded separately); stale Git-state and test-database-default claims corrected (`CLAUDE.md`, `docs/09`, `docs/15`); `backend/reset_hash.go` plaintext-password publication risk added to `docs/08` and `docs/12`; sprint/SDLC records extended with the 2026-10-28 sessions; safe `.env` creation instructions added to the README; execution log and prompt log updated.
- The supplied `README(2).md` was not found in the project — its comparison against the root README is **BLOCKED** pending placement at `docs/README-reference.md`.

## 2026-10-28

### Added
- **Pending-registration notifications for administrators.** New backend endpoints `GET /api/registrations` and `POST /api/registrations/{id}/dismiss` (admin-only, workspace-scoped): administrators see registered users who have no workspace membership (name, email, registration date, status) behind a notification bell with a live count; "Add to Team" reuses the existing membership API; "Dismiss" hides a registration for that workspace only. Migration `0002_pending_registration_dismissals` adds the `dismissed_registrations` table (additive, forward-only). Existing member removal, revocation, and task-reassignment behavior unchanged.
- New integration test `TestPendingRegistrations` (9 subtests) + E2E suite `registrations_e2e.py` (19 checks) and restart-persistence check (3 checks).

### Changed
- Integration tests now default `TEST_DATABASE_URL` to a dedicated `task_management_tests` database (create once with `CREATE DATABASE task_management_tests OWNER taskapp;`) so running the test suite never truncates application data.

## 2026-10-27

### Fixed
- **Backend could not boot against PostgreSQL**: the starter applied the multi-statement `schema.sql` through the pgx pool's `Exec` (extended protocol rejects multi-statement SQL). Replaced with a transactional migration runner (`store.Migrate`) over embedded `migrations/*.up.sql`, tracked in `schema_migrations`.
- **Administrator status-only task updates were broken** (the UI status dropdown sent `{"status":…}` and the backend demanded a full payload → 400). `PUT /api/tasks/{id}` now distinguishes status-only updates (assignee or admin) from full administrative edits.
- **Member removal data integrity**: tasks of a removed member stayed assigned to a non-member and vanished from lists/dashboard. Removal now reassigns their tasks to the administrator in the same transaction.
- **`.env` was never loaded** (no dotenv in the stack) — README setup instructions could not work. Added a stdlib `.env` loader; real environment variables take precedence.
- **Project reschedules could strand task due dates** outside the project range; now rejected with 409.
- First-registration race (two simultaneous "first" users) — serialized via `pg_advisory_xact_lock`.
- Login user-enumeration timing oracle — dummy bcrypt comparison on unknown emails.
- Dashboard silently swallowed DB errors (returned zeros); now 500s with a generic message.
- `addMember` returned misleading 201 "added" for existing members; now explicit 409s.
- `store.IsNoRows` compared the wrong sentinel (`sql.ErrNoRows` vs pgx); unused helpers removed; `go.mod` bumped to 1.24 (for `t.Chdir` in tests); frontend: added confirm-password match check, 401 auto-logout, `VITE_API_URL` support, removed dead duplicated select markup.

### Added
- Migration system: `backend/internal/store/migrations/0001_init.up.sql` + ledger; `backend/db/schema.sql` kept as the reference schema.
- Test suite: 21-subtest integration workflow (real PostgreSQL, skips when unreachable), 10 unit tests (tasks/projects validation, JWT round-trip, email rules, strict decode, auth middleware, `.env` loader), scripted 44-check live-server E2E.
- Documentation set `docs/01`–`docs/16` (assignment overview, requirements, user stories, architecture, prompt log, test plan, sprint plan, known issues, SDLC deliverables, API reference, database design, security review, acceptance checklist, execution commands, Claude Code usage, demo script), `CLAUDE.md`, this changelog; legacy `docs/API.md` and `docs/ACCEPTANCE_CHECKLIST.md` now point at the canonical `10`/`13` documents.

### Verified (all PASS on 2026-10-27)
`gofmt` clean · `go vet ./...` · `go test ./...` · `go build ./...` · `npm ci` · `npm run build` · live E2E 44/44 · DB constraint probes (UNIQUE/CHECK/FK/cascade) · persistence across restart · `go test -race` BLOCKED (no cgo/gcc; see docs/08).
