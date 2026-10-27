# 13 — Acceptance Checklist

The 15 assignment acceptance criteria plus the expanded delivery checklist, each with verification evidence.
Verification date: 2026-10-27 · Go 1.27.3 · PostgreSQL 17.6 · Node 24 · raw command output in [14-execution-commands.md](14-execution-commands.md).

Legend: **PASS** = verified by an executed command/test · **FAIL** / **BLOCKED** = none outstanding except where noted.

## The 15 acceptance criteria

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | Registration and login with secure password storage and token auth | **PASS** | bcrypt hashing in `register`; HS256 expiring JWT; e2e "register first user", "admin login"; `TestMakeTokenRoundTrip` |
| 2 | Email addresses unique and properly validated | **PASS** | `users_email_key` UNIQUE (SQL-verified); regex + normalization; e2e duplicate → 409; bad format → 400 |
| 3 | Password and confirm must match | **PASS** | Backend 400 (e2e "password confirm mismatch"); frontend match check in `App.tsx` `auth()` |
| 4 | Protected pages and APIs reject unauthenticated requests | **PASS** | e2e 401 checks (no token, garbage token); `TestAuthMiddlewareRejectsMissingAndGarbageTokens`; UI gates on session |
| 5 | Only Administrator adds/removes members and manages projects | **PASS** | e2e "member cannot add/remove members / create project" → 403; `admin` middleware |
| 6 | Tasks assignable only to workspace members | **PASS** | e2e "assignee not in workspace → 400"; integration outsider-assignee + cross-workspace reassign checks (INSERT…SELECT membership join) |
| 7 | Project end date not earlier than start date | **PASS** | e2e "project end before start → 400"; DB `projects_check` (SQL-verified) |
| 8 | Task due date within project date range | **PASS** | e2e "task due date outside project range → 400"; integration edit-outside-range 400; project reschedule stranding due dates → 409 |
| 9 | Required fields validated on the frontend before submission | **PASS** | `required`/`minLength` on all auth + modal fields; password-match check; no request sent when invalid |
| 10 | Backend validates incoming data independently | **PASS** | `TestValidateTask`, `TestValidProject`; every e2e 400 case (name lengths, enums, date formats, unknown fields rejected) |
| 11 | Members cannot modify tasks assigned to other members | **PASS** | e2e + integration 403 checks (status update on another's task; full edit) |
| 12 | Project deletion leaves no orphaned tasks | **PASS** | FK `ON DELETE CASCADE`; e2e "no orphaned tasks after project delete"; SQL cascade check left 0 rows |
| 13 | Search/filter and dashboard counts accurate | **PASS** | Integration filter matrix + dashboard counts vs seeded data (admin & member); e2e counts incl. overdue |
| 14 | Success and API failure messages displayed | **PASS** | Notice/error banners + modal error display in `App.tsx`; all backend errors surfaced as `{"error"}` and rendered |
| 15 | PostgreSQL for persistent storage | **PASS** | Persistence-across-restart check (seed row → restart → served via API); migrations ledger `schema_migrations` |

## Expanded delivery checklist

- [x] Registration validates name/email/password + confirmation — e2e + integration
- [x] Duplicate email rejected — 409 e2e; UNIQUE constraint SQL-verified
- [x] Passwords bcrypt-hashed, never plaintext — code review + schema
- [x] Login issues signed, expiring JWT — `TestMakeTokenRoundTrip`
- [x] Protected APIs reject missing/invalid/expired JWTs — 401 checks; per-request membership re-check
- [x] First account = administrator of own workspace; race-safe (advisory lock) — integration subtest
- [x] Later accounts have no access until invited by exact email — integration + e2e
- [x] Admin adds/removes members; self-removal blocked — e2e
- [x] Removed member loses access immediately (unexpired JWT included) — integration + e2e; tasks reassigned
- [x] Only admin manages projects — e2e 403s
- [x] Project end ≥ start — app + DB CHECK
- [x] Task assignee must belong to project's workspace — e2e + integration
- [x] Task due date inside project range (create, edit, and project reschedule) — e2e + integration
- [x] Frontend required-field and date validation — `required`/`min`/`minLength` + confirm-password check
- [x] Backend independent validation — strict decode, enums, lengths, date parsing
- [x] Members see only their tasks; status-only updates on own tasks — integration
- [x] Filters by project/status/priority accurate — integration matrix + e2e
- [x] Dashboard totals/pending/completed/overdue correct; overdue excludes completed; member-scoped — integration + e2e
- [x] Project deletion cascades — e2e + SQL verification
- [x] Frontend success/failure messages — banners; verified in code paths exercised by e2e
- [x] `gofmt` clean · `go vet ./...` PASS · `go test ./...` PASS · `go build` PASS
- [x] `npm ci` (lockfile) PASS · `npm run build` PASS
- [x] DB constraints + migration ledger verified; persistence across restart verified
- [x] README verified against actual startup on a fresh database
- [ ] GitHub repository published — **BLOCKED on user action** (no git repo/push authorized yet; nothing fabricated)

## Legacy checklist
The earlier `docs/ACCEPTANCE_CHECKLIST.md` now points here; its items are all covered above.
