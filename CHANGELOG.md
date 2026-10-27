# Changelog

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
