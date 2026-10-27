# 11 — Database Design

PostgreSQL 15+. Schema applied by the embedded migration `backend/internal/store/migrations/0001_init.up.sql` (tracked in `schema_migrations`); `backend/db/schema.sql` is the identical human-readable reference.

## Entity relationships
```
users 1 ─── 1 workspaces (admin_user_id, UNIQUE)
users 1 ─── n workspace_members n ─── 1 workspaces   (one row per user: UNIQUE(user_id))
workspaces 1 ─── n projects n ─── 1…n tasks
users 1 ─── n tasks (assigned_user_id)
```

## Tables

### users
| Column | Type | Constraints |
|---|---|---|
| id | BIGSERIAL | PK |
| name | TEXT | NOT NULL, CHECK length(trim) 2–120 |
| email | TEXT | NOT NULL, **UNIQUE** |
| password_hash | TEXT | NOT NULL (bcrypt) |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

### workspaces
| Column | Type | Constraints |
|---|---|---|
| id | BIGSERIAL | PK |
| name | TEXT | NOT NULL |
| admin_user_id | BIGINT | NOT NULL, **UNIQUE**, FK → users ON DELETE RESTRICT |

### workspace_members
| Column | Type | Constraints |
|---|---|---|
| workspace_id | BIGINT | FK → workspaces ON DELETE CASCADE |
| user_id | BIGINT | FK → users ON DELETE CASCADE, **UNIQUE** (one workspace per user) |
| role | TEXT | CHECK in ('Administrator','Member') |
| joined_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| — | — | PK (workspace_id, user_id) |

### projects
| Column | Type | Constraints |
|---|---|---|
| id | BIGSERIAL | PK |
| workspace_id | BIGINT | NOT NULL, FK → workspaces ON DELETE CASCADE |
| name | TEXT | NOT NULL, CHECK length(trim) 2–160 |
| description | TEXT | NOT NULL DEFAULT '' |
| start_date / end_date | DATE | NOT NULL, CHECK end_date ≥ start_date |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

### tasks
| Column | Type | Constraints |
|---|---|---|
| id | BIGSERIAL | PK |
| project_id | BIGINT | NOT NULL, FK → projects **ON DELETE CASCADE** (no orphans) |
| title | TEXT | NOT NULL, CHECK length(trim) 2–200 |
| description | TEXT | NOT NULL DEFAULT '' |
| assigned_user_id | BIGINT | NOT NULL, FK → users ON DELETE RESTRICT |
| priority | TEXT | CHECK in ('Low','Medium','High') |
| due_date | DATE | NOT NULL (range-vs-project enforced in app layer) |
| status | TEXT | NOT NULL DEFAULT 'To Do', CHECK in ('To Do','In Progress','Completed') |
| created_at / updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

### schema_migrations
`version TEXT PK`, `applied_at TIMESTAMPTZ` — migration ledger written transactionally with each applied file.

## Indexes
`idx_projects_workspace(projects.workspace_id)`, `idx_tasks_project(tasks.project_id)`, `idx_tasks_assignee(tasks.assigned_user_id)`, `idx_tasks_status_priority(tasks.status, priority)` — supporting the workspace scoping, dashboard aggregation, and filter paths.

## Design decisions
- **`UNIQUE (workspace_members.user_id)`** — the assignment models one workspace per user; the DB makes cross-workspace membership impossible, not just the API.
- **Cascade policy** — deleting a workspace/project removes dependents (no orphans); deleting a *user* is RESTRICTed via tasks (the app never deletes users; member removal deletes only the membership row and reassigns tasks).
- **Enum-like CHECKs** — priority/status/date-range rules enforced twice (app validation for good messages, DB CHECKs as the last line of defense).
- **Task due-date range** — enforced at task create/edit in SQL (`due_date BETWEEN project.start_date AND end_date` inside the INSERT/UPDATE) and project reschedules that would strand a due date are rejected (409) before update.

## Constraint verification (2026-10-27, direct SQL)
Every rule above was exercised and raised the expected error: `users_email_key`, `projects_check`, `tasks_priority_check`, `tasks_status_check`, `tasks_project_id_fkey`, `tasks_assigned_user_id_fkey`, `workspace_members_user_id_key`, `workspaces_admin_user_id_key`; project DELETE left 0 tasks. Full command log: [14-execution-commands.md](14-execution-commands.md).
