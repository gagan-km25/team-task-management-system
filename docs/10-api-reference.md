# 10 — API Reference

Base URL: `http://localhost:8080` (configurable via `PORT`). All bodies and responses are JSON. Errors use `{"error":"..."}`. All `/api/*` routes except registration and login require `Authorization: Bearer <JWT>` (HS256, `uid`/`wid`/`role` claims, expiry `JWT_TTL_HOURS`).

CORS is enabled only for the configured `FRONTEND_ORIGIN` with methods GET/POST/PUT/DELETE/OPTIONS and headers Authorization/Content-Type.

## Health
### `GET /health` → 200
`{"status":"ok"}`

## Authentication
### `POST /api/auth/register` → 201
Body: `{"name":"…","email":"…","password":"…","confirmPassword":"…"}`.
- Name 2–120 chars, valid email, password ≥ 8 chars, confirmation must match; email normalized (lowercase/trimmed) and unique.
- **First ever user**: a workspace is created, role Administrator, response includes `token` and `user{…}` and the caller is signed in.
- **Later users**: account created with no workspace; response has `message` and `userId`, role `"Unassigned"`, **no token**. They gain access only when an administrator adds their exact email.
- Errors: 400 validation (including confirmation mismatch), 409 duplicate email.

### `POST /api/auth/login` → 200
Body: `{"email":"…","password":"…"}` → `{"token":"…","user":{"id","name","email","role","workspaceId"}}`.
- 401 invalid credentials (identical message and timing whether or not the email exists).
- 403 when the account has no workspace membership (registered but not yet added).

### `GET /api/me` → 200
Current identity: `{"id","name","email","role","workspaceId"}` from the live membership check.

## Dashboard
### `GET /api/dashboard` → 200
`{"totalProjects","totalTasks","completedTasks","pendingTasks","overdueTasks"}`
- Administrator: whole workspace. Member: projects with tasks assigned to them and only their tasks. Overdue = not completed and `due_date < CURRENT_DATE`.

## Members
### `GET /api/members` → 200
`[{"id","name","email","role"},…]` for the caller's workspace, ordered by name.

### `POST /api/members` → 201 (Administrator)
Body: `{"email":"…"}` — exact registered email (normalized). Errors: 400 invalid email, 404 not registered, 409 already a member of this workspace or belongs to another workspace.

### `DELETE /api/members/{id}` → 200 (Administrator)
Removes the membership and, in the same transaction, reassigns the member's tasks to the administrator. 400 when targeting self; 404 when not a Member of this workspace. The removed user's existing JWTs stop working on the next request.

## Projects
### `GET /api/projects` → 200 — caller's workspace, newest first.
### `POST /api/projects` → 201 (Administrator)
Body: `{"name","description","startDate","endDate"}` — name 2–160, dates `YYYY-MM-DD`, `endDate ≥ startDate`.
### `GET /api/projects/{id}` → 200 — 404 unless in caller's workspace.
### `PUT /api/projects/{id}` → 200 (Administrator)
Same validation; **409** if existing task due dates would fall outside the new range (reschedule those tasks first).
### `DELETE /api/projects/{id}` → 200 (Administrator)
Tasks cascade-deleted by FK; never orphans tasks.

## Tasks
### `GET /api/tasks?projectId=&status=&priority=` → 200
Workspace-scoped; members receive only tasks assigned to them. Filter values are validated (400 on invalid). Each item: `{"id","projectId","projectName","title","description","assignedUserId","assigneeName","priority","dueDate","status"}`.

### `POST /api/tasks` → 201 (Administrator)
Body: `{"projectId","title","description","assignedUserId","priority","dueDate","status"}`.
- Title 2–200, priority Low/Medium/High, status To Do/In Progress/Completed (defaults To Do), due date `YYYY-MM-DD`.
- Assignee must be a member of the caller's workspace; due date must lie within the project's range — both enforced in the INSERT … SELECT; failure → 400.

### `PUT /api/tasks/{id}` → 200 — two accepted shapes
- **Status-only** `{"status":"…"}`: permitted for the task's assignee and administrators (this is what the UI status dropdown sends). Only `status` changes.
- **Full body** (all fields): Administrator only; re-validates project scope, assignee membership, and due-date range. 400 on rule violations; members sending a full body get 403.
- 404 when the task is not in the caller's workspace.

### `DELETE /api/tasks/{id}` → 200 (Administrator)

## Status code summary
400 validation · 401 missing/invalid/expired token · 403 authenticated but not permitted (role, ownership, or no workspace) · 404 not found or outside workspace (indistinguishable by design) · 409 conflicts (duplicate email, membership conflicts, stranded due dates) · 5xx unexpected internal failures with generic messages.
