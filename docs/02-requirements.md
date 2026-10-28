# 02 — Requirements

Source of truth: the assignment document (see [01-assignment-overview.md](01-assignment-overview.md)). Status reflects verification on 2026-10-27; evidence in [13-acceptance-checklist.md](13-acceptance-checklist.md) and [14-execution-commands.md](14-execution-commands.md).

## Functional requirements

### 1. User Registration and Authentication
| # | Requirement | Status |
|---|---|---|
| 1.1 | Registration page with name, email, password, confirm password | DONE — `frontend/src/App.tsx` auth form; backend `register` handler |
| 1.2 | Registered users can log in and log out | DONE — `POST /api/auth/login`; logout clears the client session |
| 1.3 | First registered user becomes Administrator of their own workspace | DONE — decided inside a transaction serialized by a Postgres advisory lock |
| 1.4 | Later users join only when the administrator adds them by exact email | DONE — `POST /api/members` exact-match lookup after normalization |
| 1.5 | Non-members cannot access workspace projects or tasks | DONE — every authenticated request re-checks workspace membership |

### 2. Team Member Management
| # | Requirement | Status |
|---|---|---|
| 2.1 | Search existing registered user by exact email and add | DONE — exact (normalized) match; 404 when not registered |
| 2.2 | Display all members with name, email, role | DONE — `GET /api/members` + Team page |
| 2.3 | Administrator removes a member | DONE — `DELETE /api/members/{id}`; admin cannot remove self |
| 2.4 | Removed members lose workspace access | DONE — membership re-checked per request; their tasks are reassigned to the admin in the same transaction |

### 3. Project Management
| # | Requirement | Status |
|---|---|---|
| 3.1 | Administrator creates, views, updates, deletes projects | DONE — all scoped to the administrator's workspace |
| 3.2 | Project has name, description, start date, end date | DONE — validated in app and by DB CHECK |
| 3.3 | All projects displayed in a list | DONE — Projects page |
| 3.4 | Only the Administrator manages projects | DONE — `admin` middleware on writes |

### 4. Task Management
| # | Requirement | Status |
|---|---|---|
| 4.1 | Administrator creates tasks under a specific project | DONE |
| 4.2 | Task has title, description, assignee, priority, due date, status | DONE — priority Low/Medium/High, status To Do/In Progress/Completed (app + DB CHECKs) |
| 4.3 | Administrator edits, reassigns, deletes tasks | DONE — full-body PUT; assignment rules re-validated |
| 4.4 | Members view tasks assigned to them and update their status | DONE — member task list filtered by assignee; status-only PUT permitted for assignee |
| 4.5 | Filtering by project, status, priority | DONE — server-side SQL filters (also surfaced in the UI) |

### 5. Dashboard
| # | Requirement | Status |
|---|---|---|
| 5.1 | Total projects, total tasks, completed, pending | DONE — `GET /api/dashboard` |
| 5.2 | Overdue tasks shown separately; excludes completed | DONE — `status <> 'Completed' AND due_date < CURRENT_DATE` |
| 5.3 | Members see summaries based on their assigned tasks | DONE — member queries scoped by `assigned_user_id` |

## Acceptance criteria (summary)
Statuses use the scheme PASS / FAIL / PARTIAL / BLOCKED / NOT VERIFIED with evidence in [13-acceptance-checklist.md](13-acceptance-checklist.md). Frontend display behaviors verified at API level and by the user's manual testing (2026-10-29); automated browser testing has not been run.

1. Secure password storage (bcrypt) + JWT auth — **PASS**
2. Unique, validated emails — **PASS** (UNIQUE constraint + regex + normalization)
3. Password/confirm must match — **PASS** (frontend check + backend rejection)
4. Protected pages/APIs reject unauthenticated requests — **PASS**
5. Only Administrator adds/removes members and manages projects — **PASS**
6. Tasks assigned only to workspace members — **PASS**
7. Project end date not earlier than start date — **PASS** (app + DB CHECK)
8. Task due date within project date range — **PASS** (create/edit enforced; reschedules that would strand tasks rejected)
9. Required fields validated on the frontend — **PARTIAL** (validation logic implemented and exercised in code; automated browser verification not run)
10. Backend validates incoming data independently — **PASS**
11. Members cannot modify others' tasks — **PASS**
12. Project deletion leaves no orphaned tasks — **PASS** (FK ON DELETE CASCADE, verified at DB level)
13. Search/filter and dashboard counts accurate — **PASS** (live-verified against database records 2026-10-28)
14. Success/failure messages in the frontend — **PARTIAL** (backend error contract verified live; UI rendering verified only by user-reported manual testing, not automation)
15. PostgreSQL persistence — **PASS** (verified across restart)

## Enhancement requirements (2026-10-28): pending registrations
| # | Requirement | Status |
|---|---|---|
| E1 | New registrations visible to the Administrator without granting access | **PASS** — derived workspace-less list; `GET /api/registrations` |
| E2 | Notification bell with pending count; details (name, email, date, status) | **PASS** — `BellPanel` in `App.tsx`; count derived from the same query (UI rendering manually verified by user) |
| E3 | Search by email/name; Add to Team via existing membership API | **PASS** — client-side filter; `POST /api/members` reused |
| E4 | Per-workspace persistent dismissal; account never deleted; no automatic reappearance | **PASS** — `POST /api/registrations/{id}/dismiss`; migration `0002`; restart-persistence verified |
| E5 | Admin-only, workspace-scoped; duplicates impossible | **PASS** — `auth`+`admin` middleware; list derived from current data |
| E6 | Loading, error-with-retry, and empty states | **PASS** (code) / UI rendering — **PARTIAL** (manual only) |

Per-criterion evidence: [13-acceptance-checklist.md](13-acceptance-checklist.md).
