# 14 — Execution Commands

Actual commands run on 2026-10-27 and the observed results. Paths: `GO` = `%USERPROFILE%\tools\go\bin`, `PG` = `%USERPROFILE%\tools\pgsql\bin` (user-local toolchain installed for this project because Go and PostgreSQL were not preinstalled; adjust to your install location).

## Final submission readiness audit (2026-10-27, later the same day)

| Check (cwd as noted) | Observed |
|---|---|
| `ls docs/` | All 16 numbered docs present (01–16) plus `ACCEPTANCE_CHECKLIST.md`, `API.md` |
| `git status` | `fatal: not a git repository` — confirmed; no commit/push performed |
| Wording scan for "committed" | 2 inaccurate lockfile claims found → corrected in `CLAUDE.md`, `docs/09-sdlc-deliverables.md` |
| `.env` file search (`find . -name ".env*"`) | Only `backend/.env.example` (placeholder values); no real `.env` in tree |
| Secret-pattern scan (long values after key/secret/password/token) | No matches in source/docs |
| `.gitignore` | Covered `.env*`, `node_modules/`, `dist/`, logs; extended with `pgdata/`, `*.dump`, `*.sql.gz`, `*.key`, `*.pem`, `*.p12` |
| Relative-link check (Python walker over all `.md`) | 59 links checked, all resolve |
| Server boot per README env vars (`DATABASE_URL`, `JWT_SECRET`, `PORT`) against `task_management_e2e` | `API listening on http://localhost:8080`; `/health` → `{"status":"ok"}`; login 200; `GET /api/dashboard` 200; migrations re-applied clean |
| `gofmt -l .` (backend) | clean |
| `go vet ./...` | pass |
| `go test -count=1 ./...` (fresh, uncached, live PostgreSQL) | `ok internal/api 5.473s` · `ok internal/config 1.126s` |
| `go build ./...` | pass |
| `npm ci` (frontend, lockfile present) | pass (benign esbuild postinstall warning) |
| `npm run build` | pass — `index.js 173.72 kB / gzip 53.83 kB` |
| `go test -race ./...` | not re-attempted: windows/amd64 race detector requires cgo and no gcc toolchain exists (original error recorded earlier in this file) — stays BLOCKED |
| Disk impact | `df -h /c` before 354 MB free → after audit ~4.0 GB free (npm ci / builds net-neutral; external space also freed during the session) |

---

## Backend

| Command (cwd `backend/`) | Result |
|---|---|
| `go mod tidy` | Created `go.sum`; downloaded jwt v5.2.1, pgx v5.7.2, x/crypto v0.31.0 + deps |
| `gofmt -l .` (before) | Listed all 9 backend `.go` files (non-canonical formatting) |
| `gofmt -w .` then `gofmt -l .` | No output → all files canonical |
| `go vet ./...` | Exit 0, no findings (run 3× across the session, incl. post-format) |
| `go test ./...` | `ok internal/api 3.358s` · `ok internal/config 0.793s` (21 integration subtests + 10 unit tests, all PASS; integration requires reachable PostgreSQL and skips otherwise) |
| `go test -race ./...` | **Failed: `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`** — no gcc on machine; mingw-w64 does not fit on the ~100%-full drive. Recorded as BLOCKED in [08-known-issues.md](08-known-issues.md) |
| `go build ./...` and `go build -o $TEMP/ttms-api.exe ./cmd/api` | Exit 0 |
| Server start | `DATABASE_URL=postgres://taskapp:…@localhost:54329/task_management_e2e?sslmode=disable JWT_SECRET="<32+ chars>" PORT=8080 ./ttms-api.exe` → log: `API listening on http://localhost:8080`; migrations applied on first boot and re-applied as a no-op on restart |
| `curl -s http://localhost:8080/health` | `{"status":"ok"}` |
| `curl -s -X OPTIONS … -H "Origin: http://localhost:5173"` | `HTTP/1.1 204`, correct `Access-Control-Allow-*` headers |
| Live E2E `python -I e2e.py` (44 checks over real HTTP) | **44 passed, 0 failed** — run twice (pre- and post-gofmt binary) |

## Frontend

| Command (cwd `frontend/`) | Result |
|---|---|
| `npm install --no-audit --no-fund` | Created `package-lock.json` and `node_modules` |
| `npm ci --no-audit --no-fund` | Clean install from lockfile (benign esbuild postinstall warning) |
| `npm run build` (`tsc -b && vite build`) | PASS — `dist/assets/index-*.js 173.72 kB / gzip 53.83 kB` (run 4× incl. post-`npm ci`) |

## Database (PostgreSQL 17.6 on localhost:54329)

| Check (psql, direct SQL) | Observed |
|---|---|
| `SELECT version();` | PostgreSQL 17.6, MSVC 64-bit |
| `SELECT version FROM schema_migrations;` | `migrations/0001_init.up.sql` |
| INSERT duplicate email | `ERROR: duplicate key … users_email_key` |
| INSERT project end<start | `ERROR: … projects_check` |
| INSERT task priority 'Urgent' | `ERROR: … tasks_priority_check` |
| INSERT task status 'Done' | `ERROR: … tasks_status_check` |
| INSERT task → missing project / missing assignee | `ERROR: … tasks_project_id_fkey` / `tasks_assigned_user_id_fkey` |
| Second membership for same user | `ERROR: duplicate key … workspace_members_user_id_key` |
| Second workspace for same admin | `ERROR: duplicate key … workspaces_admin_user_id_key` |
| DELETE project with tasks | task count after delete = **0** (cascade) |
| Persistence: INSERT row → stop server → start server → login + `GET /api/projects` | Row returned; migration re-run clean |

## Environment setup (one-time, not part of the repo)
- Go 1.27.3 zip from go.dev extracted to `%USERPROFILE%\tools\go`.
- PostgreSQL 17.6 EDB binaries (bin/lib/share) extracted to `%USERPROFILE%\tools\pgsql`; `initdb` → `%USERPROFILE%\tools\pgdata`; `postgres.exe` started directly on port 54329 (pg_ctl detach misbehaves under Git Bash); roles `postgres` (trust, local dev) and `taskapp`; databases `task_management` (tests) and `task_management_e2e` (live runs).
- Disk maintenance: cleared npm/pip caches, Go build cache, and extracted archives twice — the system drive is ~100% full (user action required).
