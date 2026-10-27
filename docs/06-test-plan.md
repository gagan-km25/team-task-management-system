# 06 — Test Plan

All automated tests ran green on 2026-10-27 (PostgreSQL 17.6, Go 1.27.3, Node 24). Commands and raw results: [14-execution-commands.md](14-execution-commands.md).

## Layers

### 1. Unit tests (no database) — `backend/internal/api/unit_test.go`, `backend/internal/config/config_test.go`
| Test | Covers |
|---|---|
| `TestHealthRoute` | JSON helper / status codes |
| `TestValidateTask` | task field rules: required project/assignee/title, priority and status enums, date format, status default |
| `TestValidProject` | project name/description/date validation, trimming, end ≥ start |
| `TestEmailPattern`, `TestNormalizeEmail` | email format + case/space normalization |
| `TestDecodeRejectsUnknownFields` | strict request-body decoding |
| `TestMakeTokenRoundTrip` | JWT claims, expiry, short-secret rejection, cross-secret rejection |
| `TestAuthMiddlewareRejectsMissingAndGarbageTokens` | 401 paths before any DB access |
| `TestApplyDotEnv`, `TestLoadDefaults` | `.env` parsing, precedence of real environment, defaults |

### 2. Integration tests (real PostgreSQL) — `backend/internal/api/integration_test.go`
Single ordered workflow (`TestIntegrationWorkflow`, 21 subtests) against a database reset per run; skips with an explanatory message when `TEST_DATABASE_URL` is unreachable. Covers: first-user administrator creation; registration validation (duplicate email, mismatch, bad email, short password); workspace-less login rejection; 401 for missing/garbage tokens; login (`/api/me`, wrong password); member add by exact email (+404 unknown, +409 duplicates); member login/role; member authorization failures (projects, members); project validation and creation; **second-workspace isolation** (404 cross-workspace read, empty task list, assignment into another workspace rejected); task validation (due date outside range, bad priority, outsider assignee, member create 403); admin/member task visibility; filters (status, priority, project, combined, invalid); member permission matrix (403 on others' tasks, 403 full edit, own status update persisted, invalid status 400); admin status-only update (regression test for the dropdown defect); dashboard counts (admin + member-scoped, overdue excluding completed); admin full edit/reassign (including out-of-range due and outsider reassign rejected); project reschedule that would strand due dates (409) and valid reschedule; task delete authorization; **member removal → access revoked + tasks reassigned** (DB-verified); project delete cascade (DB-verified zero orphans).

### 3. Scripted end-to-end (live server) — `e2e.py` (kept outside the repo; 44 checks)
Runs the real binary over real HTTP: registration flows, login, role gates, member management, project/task CRUD and validation, status-only updates, filters, dashboard counts including overdue and member scoping, removal revocation, cascade delete. **Result: 44 passed, 0 failed** (re-run after final formatting).

### 4. Database constraint checks (direct SQL via psql)
Verified each raises the expected error: duplicate email (`users_email_key`), project `end < start` (`projects_check`), invalid priority/status CHECKs, task FK to missing project (`tasks_project_id_fkey`) and missing assignee (`tasks_assigned_user_id_fkey`), one-workspace-per-user (`workspace_members_user_id_key`), one-workspace-per-admin (`workspaces_admin_user_id_key`), and project delete cascade leaving zero tasks.

### 5. Persistence across restart
Row seeded via SQL → server stopped → server restarted → fresh login + `GET /api/projects` returns the row; migration re-run is a clean no-op.

## Gaps
- `go test -race` requires cgo (gcc) which is unavailable on this machine — see [08-known-issues.md](08-known-issues.md).
- No browser-driven UI automation; the UI is verified by the production build compiling and by API-level tests of every operation the UI performs.
