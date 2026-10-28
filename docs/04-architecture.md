# 04 — Architecture

## Overview
```
┌─────────────────────────┐        HTTP/JSON + JWT          ┌──────────────────────────────┐
│  React 18 + Vite (TS)   │  ─────────────────────────────▶ │  Go REST API (net/http)      │
│  functional components, │   Authorization: Bearer <JWT>   │  cmd/api → internal/api      │
│  hooks, Vite dev/build  │ ◀─────────────────────────────  │  handlers → store (pgx v5)   │
└─────────────────────────┘         JSON responses          └──────────────┬───────────────┘
                                                                           │ parameterized SQL
                                                            ┌──────────────▼───────────────┐
                                                            │  PostgreSQL 15+              │
                                                            │  users, workspaces,          │
                                                            │  workspace_members,          │
                                                            │  projects, tasks             │
                                                            └──────────────────────────────┘
```

## Backend layering
- `cmd/api/main.go` — configuration load, DB open, migrations, HTTP server with graceful shutdown.
- `internal/config` — environment + `.env` loading (shell environment wins).
- `internal/store` — pgx pool, embedded SQL migrations with a `schema_migrations` ledger, small helpers.
- `internal/api` — route table (Go 1.22+ method+path patterns), CORS for the configured frontend origin, and handlers:
  - `auth middleware` — verifies the HS256 JWT (algorithm pinned in the key function), then re-reads the caller's **current** workspace membership from the database, so revocation is immediate.
  - `admin middleware` — role gate for administrator-only routes, including the pending-registration endpoints `GET /api/registrations` and `POST /api/registrations/{id}/dismiss`.
  - Handlers keep every query scoped by `workspace_id` derived from the verified identity, never from client input.

## Request flow (protected route)
1. `Authorization: Bearer` extracted; missing → 401.
2. JWT parsed with HS256 pinned; invalid/expired → 401.
3. `users ⋈ workspace_members` lookup for the token subject + workspace → absent → 403.
4. Handler executes parameterized, workspace-scoped SQL.
5. JSON response; errors use `{"error":"..."}` with suitable status codes; internals are not leaked.

## Data integrity rules
- App-level: date validation, assignment rules, status-only member updates, first-user election (advisory lock `pg_advisory_xact_lock`), member-removal task reassignment (single transaction).
- DB-level: FKs with appropriate `ON DELETE` actions, CHECK constraints for enums and date ranges, `UNIQUE` on email, `admin_user_id`, and `workspace_members.user_id` (one user, one workspace), indexes on foreign keys and filter columns. See [11-database-design.md](11-database-design.md).

## Frontend structure
Single-page app (`frontend/src/App.tsx`) with:
- token + user persisted in `localStorage`; auth screen when absent.
- pages: Overview (dashboard), Projects, My Tasks, Team; filters (project/status/priority + text search); modals for project/task/member forms; notice/error banners; role-gated buttons (writes hidden for members; enforced again server-side).
- **Administrator notification bell** (`BellPanel` component): count badge derived from `GET /api/registrations`, search by email/name, Add to Team (reuses `POST /api/members`) and Dismiss (`POST /api/registrations/{id}/dismiss`), with loading, error-with-retry, and empty states; refreshed on open and after actions.
- API base from `VITE_API_URL` (default `http://localhost:8080`); 401 responses clear the session.

## Migrations
Embedded `migrations/*.up.sql` applied in filename order inside transactions at startup; applied files recorded in `schema_migrations`. Re-running is a no-op. `backend/db/schema.sql` is the reference schema for reviewers.
