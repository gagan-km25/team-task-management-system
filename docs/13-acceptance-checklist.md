# 13 — Acceptance Checklist

The 15 assignment acceptance criteria plus the enhancement and delivery checklists, each with evidence and file/function references.

Status scheme: **PASS** (implemented + executed verification) · **FAIL** · **PARTIAL** (partially satisfied or partially verified) · **BLOCKED** (documented dependency/environment constraint) · **NOT VERIFIED** (insufficient evidence).

Evidence dates: automated verification 2026-10-27/28 (Go 1.27.3 toolchain, PostgreSQL 17.6, Node 24); the user reported manual UI testing on 2026-10-29 — recorded as **user-reported**, distinct from automated evidence. Raw command output: [14-execution-commands.md](14-execution-commands.md).

## The 15 assignment acceptance criteria

| # | Criterion | Status | Evidence (test/command) | References | Limitation |
|---|---|---|---|---|---|
| 1 | Registration and login with secure password storage and token auth | **PASS** | `TestMakeTokenRoundTrip` (claims/expiry/alg/secret≥32); live E2E "register first user → 201 + token", "wrong password → 401"; bcrypt in `register` | `backend/internal/api/auth.go` (`register`, `makeToken`, `auth`); `users.password_hash` | — |
| 2 | Email addresses unique and properly validated | **PASS** | `TestEmailPattern`/`TestNormalizeEmail`; E2E duplicate → 409, invalid → 400; live constraint `users_email_key` | `auth.go` (`emailPattern`); schema `users` | — |
| 3 | Password and confirm must match | **PASS** | E2E "password confirm mismatch → 400"; frontend guard in `auth()`; `registration_validation` subtest | `auth.go` register; `App.tsx` | Browser form behavior manually verified by user (user-reported) |
| 4 | Protected pages/APIs reject unauthenticated requests | **PASS** | Live probes: 401 on projects/tasks/dashboard/members/registrations (no + garbage token); `TestAuthMiddlewareRejectsMissingAndGarbageTokens` | `auth` middleware (`auth.go`) | — |
| 5 | Only Administrator adds/removes members, manages projects | **PASS** | E2E member 403s (add/remove members, create project); `member_cannot_manage_projects_or_members` subtest; live member 403s | `admin` middleware; `members`/`project` handlers | — |
| 6 | Tasks assigned only to workspace members | **PASS** | E2E "assignee not in workspace → 400"; `task_validation` + cross-workspace reassign subtests | `createTask`/`updateTask` INSERT…SELECT membership join | — |
| 7 | Project end date not earlier than start date | **PASS** | E2E end<start → 400; live constraint `projects_check`; `TestValidProject` | `validProject` (`handlers.go`); schema `projects` | — |
| 8 | Task due date within project range | **PASS** | E2E outside-range → 400; `admin_full_edit_and_reassign` (400); `project_update_cannot_strand_task_due_dates` (409) | `createTask`/`updateTask` SQL; `updateProject` guard | — |
| 9 | Frontend validates required fields before submission | **PARTIAL** | Validation logic implemented (`required`/`minLength`/`min` attributes, password-match guard in `App.tsx`) and exercised via code review + backend 400s; automated browser test not run; user-reported manual testing 2026-10-29 | `App.tsx` forms | No automated browser evidence |
| 10 | Backend validates data independently | **PASS** | `TestValidateTask`, `TestValidProject`, `TestDecodeRejectsUnknownFields`; all E2E 400 cases | `validateTask`/`validProject`/`decode` (`handlers.go`, `api.go`) | — |
| 11 | Members cannot modify others' tasks | **PASS** | E2E member full-edit → 403; `member_task_permissions` (other's task 403, own status 200 persisted, invalid status 400) | `updateTask` status-only path | — |
| 12 | Project deletion leaves no orphaned tasks | **PASS** | `project_delete_cascades_tasks` (DB count 0); E2E "no orphaned tasks"; live FK `tasks_project_id_fkey` ON DELETE CASCADE | schema `tasks`; `deleteProject` | — |
| 13 | Search/filter and dashboard counts accurate | **PASS (live-verified)** | 2026-10-28 live check: API `{projects:4, tasks:7, completed:4, pending:3, overdue:0}` = exact SQL truth; filters Completed→4, In Progress→2, projectId→1; suite `task_filters` + `dashboard_counts` | `dashboard`/`tasks` handlers | — |
| 14 | Success/failure messages displayed in the frontend | **PARTIAL** | Backend error contract verified live on every failure path (`{"error":…}`); UI rendering implemented (`notice`/`error` banners, `BellPanel` states) and user-reported manual testing 2026-10-29; no automated browser evidence | `App.tsx` | No automated browser evidence |
| 15 | PostgreSQL for persistent storage | **PASS** | Persistence-across-restart probes (3/3, twice); `schema_migrations` ledger (0001+0002 applied to live DB); constraints verified by direct SQL | `store.Migrate`; live DB | — |

## Enhancement: pending registrations (2026-10-28)
| ID | Requirement | Status | Evidence |
|---|---|---|---|
| P1 | Registrations visible to admin; bell count; name/email/date/status | **PASS (API)** + UI user-reported | `TestPendingRegistrations`; `registrations_e2e.py` 21/21; `registrations` handler; `BellPanel` |
| P2 | Search by email/name | **PASS** (client-side filter; logic in `BellPanel`) | Browser typing — manual |
| P3 | Add to Team via existing membership API; refresh both lists | **PASS (live)** | E2E add 201 → pending−1, members+1, duplicate 409, dismiss-member 409 |
| P4 | Dismiss per-workspace, persistent, account-safe, idempotent | **PASS (live)** | E2E dismiss 200 + re-dismiss 200; `dismissed_registrations` row verified; account login still 403 (exists, workspace-less); restart-persistence 3/3 |
| P5 | Admin-only, workspace-scoped, isolation between workspaces | **PASS** | `TestPendingRegistrations` isolation subtest; live 401/403 gates |
| P6 | Loading / error-with-retry / empty states | **PARTIAL** | Implemented in `BellPanel`/`load()`; rendering verified only by user-reported manual testing |

## Delivery checklist
- [x] `gofmt -l .` clean · `go vet ./...` pass · `go test ./...` pass (isolated `task_management_tests` DB) · `go build` pass
- [x] `npm ci` + `npm run build` pass
- [x] Original live-server E2E: 44/44 (2026-10-27) · pending-registration E2E: 21/21 (2026-10-28) · restart persistence: 3/3 (2026-10-28)
- [x] Live DB: migrations 0001+0002 applied; all constraints present; dashboard/filters match SQL truth (2026-10-28)
- [ ] `go test -race ./...` — **BLOCKED**: race detector requires cgo; no gcc toolchain in this environment (docs/08 #1)
- [ ] Browser UI automation — **NOT VERIFIED**: no automation framework; manual steps in [16-demo-script.md](16-demo-script.md) (user-reported manual pass 2026-10-29)
- [ ] `README(2).md` comparison — **BLOCKED**: reference file not placed at `docs/README-reference.md` (requested 2026-10-29)

## Legacy pointers
`ACCEPTANCE_CHECKLIST.md` points here. Earlier detailed statuses are retained in git history of the parent working copy.
