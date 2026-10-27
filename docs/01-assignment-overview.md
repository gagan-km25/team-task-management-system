# 01 — Assignment Overview

## Problem statement
Develop a web-based **Team Task Management System** that allows a team administrator to manage team members, create projects, assign tasks, and monitor their progress.

## Functional requirement areas (from the assignment document)
1. **User Registration and Authentication** — registration with name/email/password/confirm; login and logout; the first registered user becomes the Administrator of their own workspace; other registered users join only when that administrator adds them by exact email; non-members cannot access workspace projects or tasks.
2. **Team Member Management** — administrator searches registered users by exact email, adds and removes members; members are listed with name, email, and role; removed members lose access immediately.
3. **Project Management** — administrator-only CRUD; each project has name, description, start date, end date; projects listed; end date cannot precede start date.
4. **Task Management** — administrator creates tasks under a project with title, description, assignee, priority (Low/Medium/High), due date, status (To Do/In Progress/Completed); administrator edits, reassigns, deletes; members view their own tasks and update status only; filters by project, status, priority.
5. **Dashboard** — total projects, total tasks, completed, pending; overdue tasks shown separately; members see summaries based on tasks assigned to them.

## Acceptance criteria
Fifteen criteria covering secure password storage and token auth, unique/valid emails, confirmation matching, rejection of unauthenticated requests, administrator-only management, workspace-scoped assignment, project and task date rules, frontend and backend validation, member task restrictions, cascade deletes, accurate search/counts, user-facing messages, and PostgreSQL persistence. The full mapped list with verification evidence lives in [13-acceptance-checklist.md](13-acceptance-checklist.md).

## Mandatory stack
| Area | Requirement |
|---|---|
| Frontend | React.js, functional components, hooks |
| Backend | Go with REST APIs |
| Database | PostgreSQL with appropriate relationships and constraints |
| Authentication | JWT-based authentication and hashed passwords |

## Submission requirements
- GitHub URL with the React frontend and Go backend source.
- PostgreSQL schema or database migrations (`backend/internal/store/migrations/`, `backend/db/schema.sql`).
- README with installation and execution instructions.
