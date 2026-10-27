# CLAUDE.md

Guidance for Claude Code sessions working in this repository.

## Project
Team Task Management System — assignment-driven. React 18 + Vite + TypeScript frontend (`frontend/`), Go REST backend (`backend/`), PostgreSQL with embedded migrations (`backend/internal/store/migrations/`). The assignment document is the source of truth; see `docs/01`–`docs/03`.

## Commands (Git Bash)
```bash
# Backend (cwd backend/) — Go is user-local on this machine
export PATH="$HOME/tools/go/bin:$PATH"   # adjust to your Go install location
gofmt -l .          # must print nothing
go vet ./...
go test ./...       # integration tests need PostgreSQL; they skip if TEST_DATABASE_URL is unreachable
go build ./...

# Frontend (cwd frontend/)
npm ci              # package-lock.json exists in the working tree (no Git repository has been initialized yet)
npm run build       # tsc -b && vite build
npm run dev         # Vite dev server (default http://localhost:5173)

# Database (dev cluster, user-local, port 54329)
export PATH="$HOME/tools/pgsql/bin:$PATH"   # adjust to your PostgreSQL install location
psql -p 54329 -U postgres -d task_management -c "…"
```

## Environment
- Integration tests default `TEST_DATABASE_URL` to `postgres://taskapp:change-this-password@localhost:54329/task_management?sslmode=disable`; override for other setups.
- Backend reads `.env` (see `backend/.env.example`); real environment variables win. `JWT_SECRET` must be ≥ 32 chars.
- The dev PostgreSQL cluster lives in a user-local data directory (this machine: `%USERPROFILE%\tools\pgdata`). If `pg_ctl start` hangs under Git Bash, start `postgres.exe` directly (see `docs/15-claude-code-usage.md`).
- The system drive is nearly full — check `df -h /c` before large builds.

## Conventions
- Match the surrounding code style; Go code must stay `gofmt`-clean.
- Every handler is workspace-scoped via the verified JWT identity (`current(r).WorkspaceID`) — never trust client-supplied workspace/role data.
- Parameterized SQL only; validate enums/dates/lengths in handlers and let DB CHECKs be the backstop.
- Any behavior change gets a test in `backend/internal/api/` (extend `integration_test.go` / `unit_test.go`).
- Update `docs/` (especially 08, 13, 14) when defects are found/fixed or verification is run; append significant user prompts to `docs/05-prompt-log.md`.
- Never commit or print secrets; never `git commit`/`push` without explicit user authorization.
