# 12 — Security Review

Reviewed 2026-10-27 against the implemented code. Scope: authentication, authorization, data integrity, input handling, secrets, and transport-adjacent concerns for this assignment's scope.

## Authentication
| Control | Implementation | Assessment |
|---|---|---|
| Password storage | bcrypt (`golang.org/x/crypto/bcrypt`, DefaultCost 10) — only the hash is stored (`users.password_hash`) | OK. Plaintext never stored or logged. |
| Password policy | ≥ 8 chars + confirmation match, enforced backend and frontend | OK for assignment scope. |
| Login | Constant error message `Invalid email or password`; dummy bcrypt comparison when the email is unknown to equalize timing | OK; prevents user enumeration via response/time differences. |
| Token | HS256 JWT, `uid`/`wid`/`role` claims, `exp` + `iat` + `iss`, TTL from `JWT_TTL_HOURS` (default 24 h) | OK. Algorithm pinned in the key function (`t.Method != jwt.SigningMethodHS256` → `ErrSignatureInvalid`) — no `alg: none`/algorithm-confusion path. Secret must be ≥ 32 chars or token issuance fails. |
| Revocation | Stateless by design; **every authenticated request re-reads workspace membership**, so removed members lose all data access immediately even with unexpired tokens (integration-verified) | Residual: a removed member's token still authenticates the *token check* but no longer maps to a membership → 403. Documented in known issues. |
| Transport | Dev runs plain HTTP on localhost; no cookies are used (Bearer header), so CSRF does not apply | Production deployment should terminate TLS at a reverse proxy. |

## Authorization
- Role gates: `admin` middleware on all member/project/task write routes; UI hides the same actions but the server never trusts that.
- Ownership gates: members may status-update only tasks where `assigned_user_id = caller`; any other field change → 403 (integration-verified).
- **Workspace scoping**: every query filters by `workspace_id` taken from the *verified* identity, never from client input. Cross-workspace IDs return 404 (indistinguishable from nonexistent — no existence leak). Integration-verified with a second workspace.
- ID manipulation: `DELETE /api/members/{id}`, `PUT /api/tasks/{id}`, etc. all join back to the caller's workspace in SQL; forged IDs cannot reach another workspace's rows (DB-verified constraints + tests).
- Membership changes: exact-email lookup; adding an admin's email to another workspace, re-adding an existing member, and targeting a member of a different workspace all 409/404 correctly.

## Input handling and injection
- 100% parameterized SQL via pgx (`$1…` placeholders); the only dynamic SQL is the task-filter builder, which appends **placeholder numbers and fixed column names** from a hardcoded allow-list — values always travel as parameters. No string interpolation of user data into SQL.
- JSON bodies decoded with `DisallowUnknownFields`; enums (priority/status/role) validated against fixed sets; lengths capped; dates parsed strictly (`2006-01-02`).
- Errors: handlers return short, generic messages (`{"error":"…"}`); SQL/stack details are never echoed. 404 doubles as "outside your workspace".
- Redirects, file uploads, HTML rendering: not present (JSON API + separate SPA), so corresponding risk classes are out of scope.

## Data integrity under concurrency
- First-user election runs inside a transaction holding `pg_advisory_xact_lock`, so simultaneous registrations cannot elect two administrators.
- Member removal + task reassignment is a single transaction (atomic; no window where tasks belong to a non-member).
- DB constraints (UNIQUE/CHECK/FK) verified to fire even if application logic regressed.

## Secrets and configuration
- `.env` is git-ignored (`!.env.example` allowed); `.env.example` contains only placeholders. JWT secret is read from the environment; short secrets are rejected at issuance.
- No secrets are hardcoded in source, tests, or docs, **with one exception flagged for removal**: the untracked local utility `backend/reset_hash.go` (not imported by the application, preserved per instruction) contains a plaintext password literal used to generate a bcrypt hash — it must be deleted or moved out of the repository before any commit/push, and the affected credential rotated. The repository itself contains no credentials; the dev DB password in `.env.example` is an obvious placeholder (`change-this-password`).
- The session's external tooling (GLM API key) was never present in project files and is not referenced anywhere.

## Findings summary
| # | Severity | Finding | Disposition |
|---|---|---|---|
| 1 | Low | JWT in `localStorage` (XSS-readable; no server-side revocation) | Accepted, documented; TTL + per-request membership check mitigate. No third-party scripts. |
| 2 | Low | No rate limiting on login/register | Accepted for assignment scope; noted for production (reverse proxy or middleware). |
| 3 | Low | Dev DB cluster uses trust auth | Local-only dev cluster; documented; not part of the deliverable. |
| 4 | Info | Member's tasks reassign to admin on removal | Product decision; prevents orphaned/invisible tasks; message states it explicitly. |

No high or critical findings remain open.
