# Team Task Management System

A full-stack team task management application: a team administrator registers, invites team members by exact email, reviews pending registrations through a notification bell, creates projects, assigns tasks, and monitors progress on a dashboard with overdue tracking.

## Implemented features
- **Registration & authentication** — name/email/password/confirm validation, bcrypt-hashed passwords, JWT (HS256) login sessions, logout. The **first registered account becomes the Administrator** of its own workspace; later accounts gain no access until the Administrator approves them by exact email.
- **Pending-registration notifications (Administrator)** — a bell with a live count badge opens a panel of registered users who have no workspace membership (name, email, registration date, "Pending" status), with **search by email or name**, **Add to Team** (existing membership API) and **Dismiss** (per-workspace, persistent, account never deleted), plus loading, error-with-retry, and empty states.
- **Team management** — add/remove members; removal immediately revokes access (membership re-checked on every request) and reassigns the removed member's tasks to the Administrator.
- **Projects** — Administrator-only CRUD with name/description/start/end validation (end date cannot precede start date; reschedules that would strand task due dates are rejected).
- **Tasks** — Administrator CRUD with title, description, assignee (workspace members only), priority (Low/Medium/High), due date (must fall inside the project range) and status (To Do/In Progress/Completed); members see their own tasks and update their status only; filters by project, status, and priority.
- **Dashboard** — total projects, total tasks, completed, pending, and overdue counts (overdue excludes completed); members see summaries scoped to their assigned tasks.

## Technology stack (verified against project files)
- **Frontend:** React 18 + Vite 6 + TypeScript (functional components and hooks) — `frontend/package.json`
- **Backend:** Go 1.24+ REST API, standard library `net/http` router, `jackc/pgx/v5` for PostgreSQL — `backend/go.mod`
- **Database:** PostgreSQL 15+ with foreign keys, CHECK constraints, indexes, and SQL migrations — `backend/internal/store/migrations/`
- **Authentication:** JWT (HS256, signed and expiring) with bcrypt-hashed passwords — `backend/internal/api/auth.go`

## Prerequisites
| Software | Version | Download |
|---|---|---|
| Go | 1.24 or newer | https://go.dev/dl/ |
| Node.js (includes npm) | 20 or newer | https://nodejs.org/ |
| PostgreSQL | 15 or newer | https://www.postgresql.org/download/windows/ |
| Git | any recent version | https://git-scm.com/download/win |

Verify in PowerShell:

```powershell
go version
node --version
npm --version
git --version
```

Each command must print a version number. If `go version` fails, Go was not added to your `PATH` — re-run the Go installer with "Add to PATH" enabled, then reopen PowerShell.

## Repository structure
```
team-task-management-system/
├── backend/
│   ├── cmd/api/                  # server entry point
│   ├── internal/api/             # routes, handlers, middleware, tests
│   ├── internal/config/          # environment/.env loader (+ tests)
│   ├── internal/store/           # pgx pool, migration runner
│   │   └── migrations/           # 0001_init, 0002_pending_registration_dismissals
│   ├── db/schema.sql             # human-readable reference schema
│   ├── .env.example              # configuration template (placeholders only)
│   ├── go.mod / go.sum
│   └── reset_hash.go             # local utility — NOT part of the app (see Security)
├── frontend/
│   ├── src/App.tsx               # application UI (hooks-based)
│   ├── src/styles.css
│   ├── package.json / package-lock.json
│   └── vite.config.ts
└── docs/                         # 16 numbered documents + pointers (see Documentation)
```

## Clone the repository

```powershell
git clone https://github.com/gagan-km25/team-task-management-system.git
cd team-task-management-system
```

All later commands assume the terminal is inside this cloned folder unless a step says otherwise.

## Step 1 — PostgreSQL setup (one time)

PostgreSQL must be installed **and running** before the backend starts (the Windows installer usually starts it as a service named `postgresql-x64-…`).

SQL statements run inside **SQL Shell (psql)**, not directly in PowerShell:
1. Open **SQL Shell (psql)** from the Start Menu.
2. Press Enter to accept the shown server (`localhost`), port (`5432`), database (`postgres`), and username — or enter a PostgreSQL administrator account such as `postgres` with the password chosen during installation.
3. At the `postgres=#` prompt:

```sql
CREATE USER taskapp WITH PASSWORD 'replace-with-a-strong-password';
CREATE DATABASE task_management OWNER taskapp;
```

Notes:
- Replace the placeholder password with your own strong password — the same value goes into the backend `.env` in Step 2.
- Skip any statement whose user/database already exists; never drop existing data.
- Leave psql with `\q`.

## Step 2 — Backend (Terminal 1)

```powershell
cd backend
if (!(Test-Path .env)) { Copy-Item .env.example .env } else { Write-Output ".env already exists - keeping it" }
notepad .env
```

The guard prevents overwriting an existing `.env`. Configure it to match Step 1:

| Variable | Required | Meaning (defaults from `backend/internal/config/config.go`) |
|---|---|---|
| `DATABASE_URL` | **Yes** | `postgres://taskapp:<your-password>@localhost:5432/task_management?sslmode=disable` |
| `JWT_SECRET` | **Yes** | Random string of **at least 32 characters**; the backend refuses to issue tokens with shorter secrets |
| `PORT` | No | Backend port (default `8080`) |
| `JWT_TTL_HOURS` | No | Login token lifetime in hours (default `24`) |
| `FRONTEND_ORIGIN` | No | Browser origin allowed by CORS (default `http://localhost:5173`) |

Start the backend:

```powershell
go mod tidy
go run ./cmd/api
```

- `go mod tidy` downloads the dependencies listed in `go.mod`/`go.sum`.
- `go run ./cmd/api` must run from the `backend` directory (`.env` is read from the current directory).
- Migrations in `backend/internal/store/migrations/` apply automatically at startup and are tracked in a `schema_migrations` table — no manual migration step.
- **Successful startup looks like:** `API listening on http://localhost:8080`.
- Health check: `curl http://localhost:8080/health` → `{"status":"ok"}`.
- **Keep this terminal open.**

## Step 3 — Frontend (Terminal 2)

Open a **second** PowerShell window:

```powershell
cd frontend
npm ci
npm run dev
```

- `npm ci` installs the exact versions in `package-lock.json` (run it inside `frontend`; the lockfile is committed).
- `npm run dev` starts Vite. Open **http://localhost:5173** (not the `127.0.0.1` URL — see Troubleshooting).
- The backend from Step 2 must be running.
- **Keep this terminal open.**

If the backend runs on a non-default port, create `frontend/.env` containing `VITE_API_URL=http://localhost:<port>` and restart `npm run dev`; the frontend otherwise calls `http://localhost:8080`.

## Startup order summary
1. PostgreSQL service running.
2. Terminal 1 — backend: `cd backend`, `go run ./cmd/api`, wait for `API listening on http://localhost:8080`.
3. Terminal 2 — frontend: `cd frontend`, `npm run dev`.
4. Browser: `http://localhost:5173`.

## First-time usage
1. **Register the first account** — it automatically becomes the **Administrator** with its own workspace.
2. Later registrations succeed but create **workspace-less accounts**; they see a message telling them an Administrator must add their email, and they cannot sign in to any workspace data.
3. As Administrator, open the **bell icon** (top bar): the **Pending registrations** panel lists waiting accounts (name, email, registration date, status), supports **search by email or name**, and shows a count badge. Use **Add to Team** to approve an account (it moves to the Team members list) or **Dismiss** to hide it for your workspace (the account is never deleted).
4. Administrators create projects/tasks and manage members on the **Team**, **Projects**, and **My Tasks** pages; members see only their assigned tasks and can update their status.
5. **Removing a member** (Team page) immediately revokes their access — even with an unexpired session — and reassigns their tasks to the Administrator.

## API summary
- `POST /api/auth/register` — name, email, password, confirmPassword
- `POST /api/auth/login` · `GET /api/me`
- `GET /api/dashboard` — totals + overdue; members get summaries scoped to their assigned tasks
- `GET /api/members`, `POST /api/members` (admin), `DELETE /api/members/{id}` (admin — revokes access, reassigns tasks)
- `GET /api/registrations`, `POST /api/registrations/{id}/dismiss` (admin) — pending-registration list and per-workspace dismissal
- `GET/POST /api/projects`, `GET/PUT/DELETE /api/projects/{id}` (writes admin-only)
- `GET /api/tasks?projectId=&status=&priority=`, `POST /api/tasks` (admin), `PUT /api/tasks/{id}`, `DELETE /api/tasks/{id}` (admin)
- `GET /health`

`PUT /api/tasks/{id}` accepts a full body (administrator edit/reassign) or a status-only body `{"status":"..."}` from the assignee or an administrator. Full details: [docs/API.md](docs/API.md) → [docs/10-api-reference.md](docs/10-api-reference.md).

## Database migrations & constraints
- Migrations live in `backend/internal/store/migrations/*.up.sql`, apply automatically in filename order at startup, run each inside a transaction, and are tracked in `schema_migrations` (re-running is a no-op). Current migrations: `0001_init` (users, workspaces, workspace_members, projects, tasks) and `0002_pending_registration_dismissals` (per-workspace dismissal records).
- Constraints enforced by PostgreSQL: unique emails, unique workspace membership per user, foreign keys with cascade rules (project deletion removes its tasks), CHECK constraints for priorities, statuses, roles, name lengths, and project date ranges.
- `backend/db/schema.sql` is the human-readable reference schema.

## Verification
```powershell
# Backend — integration tests run against TEST_DATABASE_URL, whose default is a dedicated
# task_management_tests database on localhost:54329. The test fixture TRUNCATES all tables,
# so never point TEST_DATABASE_URL at a database holding real data. Tests skip when the
# database is unreachable.
cd backend
go vet ./...
go test ./...

# Frontend
cd frontend
npm run build
```
Status per criterion: [docs/13-acceptance-checklist.md](docs/13-acceptance-checklist.md). Executed runs and observed outputs: [docs/14-execution-commands.md](docs/14-execution-commands.md).

## Troubleshooting
**`go` is not recognized.** Go is not on your `PATH` — reinstall with "Add to PATH" enabled and reopen PowerShell.

**`CREATE USER` is not recognized.** SQL belongs inside SQL Shell (psql), not PowerShell (see Step 1).

**`npm ci` reports a missing lockfile.** Run it inside `frontend`, not the repository root. If the lockfile is truly absent, re-clone; `npm install` is a fallback with slightly looser versions.

**PowerShell cannot find `frontend`.** You are not in the repository root — `cd` to the cloned folder first.

**Database connection failure.** Check `Get-Service postgresql*` is `Running`; verify `DATABASE_URL` username/password/port/database match Step 1; start the backend from the `backend` folder so it finds `.env`.

**Port already in use (`bind: Only one usage of each socket address`).** Stop the other program, or set `PORT=8081` in `backend\.env` and create `frontend\.env` with `VITE_API_URL=http://localhost:8081`, then restart both terminals.

**CORS / network errors in the browser.** Open `http://localhost:5173`, not `http://127.0.0.1:5173` — CORS allows only `FRONTEND_ORIGIN` (default `http://localhost:5173`). To use `127.0.0.1`, set `FRONTEND_ORIGIN=http://127.0.0.1:5173` in `backend\.env` and restart.

**`npm ci` fails with EPERM unlinking `esbuild.exe`.** A Vite dev server is still running and holds the file. Stop it (close the dev terminal, or `Stop-Process -Name node`), then re-run `npm ci`, then `npm run dev` again.

## Security
- Never commit `backend\.env` or real secrets; `.gitignore` excludes `.env` and key material. `backend/.env.example` contains placeholders only.
- `JWT_SECRET` must be ≥ 32 characters (enforced).
- `backend/reset_hash.go` is a local developer utility that contains a plaintext password literal; it is **not** used by the application and **must be removed before any publication** (tracked in `docs/08-known-issues.md`).
- Known limitations (JWT-in-localStorage, stateless revocation model, single-workspace-per-user, no rate limiting): [docs/08-known-issues.md](docs/08-known-issues.md) and [docs/12-security-review.md](docs/12-security-review.md).

## Git / publication status
- GitHub repository: https://github.com/gagan-km25/team-task-management-system (public, branch `main`).
- The published commit contains the original submission; the pending-registration enhancement and documentation updates are **local uncommitted working-tree changes**. Nothing has been committed or pushed since that initial publication.

## Documentation
| Doc | Content |
|---|---|
| [docs/01-assignment-overview.md](docs/01-assignment-overview.md) | Assignment problem statement and stack mandate |
| [docs/02-requirements.md](docs/02-requirements.md) | Functional requirements + criteria with status |
| [docs/03-user-stories.md](docs/03-user-stories.md) | Administrator and member stories |
| [docs/04-architecture.md](docs/04-architecture.md) | System design and request flow |
| [docs/05-prompt-log.md](docs/05-prompt-log.md) | Development prompts |
| [docs/06-test-plan.md](docs/06-test-plan.md) | Test layers and coverage |
| [docs/07-sprint-plan.md](docs/07-sprint-plan.md) | Milestones and outcomes |
| [docs/08-known-issues.md](docs/08-known-issues.md) | Verified defects and limitations |
| [docs/09-sdlc-deliverables.md](docs/09-sdlc-deliverables.md) | SDLC artifact map and quality gates |
| [docs/10-api-reference.md](docs/10-api-reference.md) | Full REST API reference |
| [docs/11-database-design.md](docs/11-database-design.md) | Schema, constraints, indexes, migrations |
| [docs/12-security-review.md](docs/12-security-review.md) | Auth, authz, input handling, findings |
| [docs/13-acceptance-checklist.md](docs/13-acceptance-checklist.md) | 15 criteria with evidence |
| [docs/14-execution-commands.md](docs/14-execution-commands.md) | Commands run and observed results |
| [docs/15-claude-code-usage.md](docs/15-claude-code-usage.md) | Tooling and workflow |
| [docs/16-demo-script.md](docs/16-demo-script.md) | Guided demo walkthrough |
