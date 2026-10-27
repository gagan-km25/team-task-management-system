# Team Task Management System

A full-stack team task management application built to the assignment requirements: an administrator manages team members, creates projects, assigns tasks, and monitors progress through a dashboard.

## Stack
- Frontend: React 18 + Vite + TypeScript (functional components and hooks)
- Backend: Go REST API (Go 1.24+, standard library `net/http` router)
- Database: PostgreSQL with foreign keys, constraints, and SQL migrations
- Authentication: JWT (HS256) with bcrypt-hashed passwords

## Prerequisites
- Go 1.24+
- Node.js 20+
- PostgreSQL 15+

## Quick start

### 1. Configure PostgreSQL
Create a database and user, for example:
```sql
CREATE USER taskapp WITH PASSWORD 'change-this-password';
CREATE DATABASE task_management OWNER taskapp;
```

### 2. Backend
```powershell
cd backend
Copy-Item .env.example .env
# Edit .env with your PostgreSQL connection string and a long JWT_SECRET (at least 32 characters)
go mod tidy
go run ./cmd/api
```
The API listens on `http://localhost:8080`. On startup it applies the SQL migrations in `backend/internal/store/migrations/` and tracks them in a `schema_migrations` table; `backend/db/schema.sql` is the human-readable reference schema. Environment variables set in the shell take precedence over `.env` values.

### 3. Frontend
In a second terminal:
```powershell
cd frontend
npm install
npm run dev
```
Open the URL printed by Vite. Set `VITE_API_URL` to point the frontend at a backend other than `http://localhost:8080`.

## First account / workspace rule
The first registered account becomes an administrator and owns a workspace. Other accounts can register but gain no workspace access until that administrator adds them by their exact registered email address. Removing a member immediately revokes their access, and their tasks are reassigned to the administrator so nothing is left assigned to a non-member.

## API
- `POST /api/auth/register` — name, email, password, confirmPassword
- `POST /api/auth/login`
- `GET /api/me`
- `GET /api/dashboard` — totals plus overdue; members get summaries scoped to their assigned tasks
- `GET /api/members`, `POST /api/members` (admin), `DELETE /api/members/{id}` (admin)
- `GET/POST /api/projects`, `GET/PUT/DELETE /api/projects/{id}` (writes admin-only)
- `GET /api/tasks?projectId=&status=&priority=`, `POST /api/tasks` (admin), `PUT /api/tasks/{id}`, `DELETE /api/tasks/{id}` (admin)
- `GET /health`

`PUT /api/tasks/{id}` accepts either a full body (administrator edit/reassign) or a status-only body `{"status":"..."}` from the assignee or an administrator. See [docs/API.md](docs/API.md).

## Verification
```powershell
# Backend (integration tests use TEST_DATABASE_URL, default localhost:54329; they skip when no database is reachable)
cd backend
go vet ./...
go test ./...

# Frontend
cd frontend
npm run build
```
The acceptance criteria and their verification status are tracked in [docs/13-acceptance-checklist.md](docs/13-acceptance-checklist.md).

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

## Important
Do not commit `.env` or real secrets. `JWT_SECRET` must be at least 32 characters.
