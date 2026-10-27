# 07 — Sprint Plan

Single-day delivery sprint (2026-10-27) following the mandated milestone order. Every milestone ended with recorded verification.

## Milestone 1 — Audit ✅
- Read every source, config, schema, doc, and test file in the starter.
- Environment bootstrap: installed user-local Go 1.27.3 and PostgreSQL 17.6 (no admin rights); created dev cluster on :54329.
- Baselines recorded: `npm run build` PASS (pre-change), `go vet`/`go test` initially blocked by missing `go.sum` (resolved via `go mod tidy`), frontend build PASS.
- Defect log produced (startup-blocking migration failure via pgx extended protocol; admin status-only update 400; member-removal integrity gap; `.env` never loaded; dashboard errors swallowed; first-user registration race; login timing oracle).

## Milestone 2 — Database and authentication ✅
- Replaced ad-hoc schema execution with a transactional, ledgered migration runner (`store.Migrate`, `schema_migrations`) and `migrations/0001_init.up.sql`.
- Fixed `.env` loading; added advisory-lock first-user election; timing-equalized login; graceful shutdown.
- Added config + auth unit tests.

## Milestone 3 — Team and projects ✅
- `addMember` rewritten (normalization, format validation, explicit already-member / other-workspace 409s).
- `removeMember` made transactional with task reassignment to the administrator.
- `updateProject` now rejects reschedules that would strand task due dates outside the range (409).
- Integration coverage for all of the above.

## Milestone 4 — Tasks and dashboard ✅
- `updateTask` redesigned: pointer-field request distinguishes status-only updates (assignee/admin) from full administrative edits — fixes the broken admin status dropdown.
- Filter inputs validated; dashboard fails loudly on DB errors; description/title length caps.
- Integration coverage: permissions matrix, filters, dashboard counts, reassignment rules.

## Milestone 5 — Frontend integration ✅
- Client-side password-confirmation match; 401 → session clear + re-auth prompt; `VITE_API_URL` support; removed dead duplicate select rendering.
- Production build (`tsc -b && vite build`) PASS after changes.

## Milestone 6 — Integration and security testing ✅
- Live-server E2E: 44/44 checks (registration → member removal → cascade).
- Direct-SQL constraint verification (UNIQUE, CHECK, FK, cascade).
- Persistence-across-restart verified. Security review written ([12](12-security-review.md)).

## Milestone 7 — Final acceptance ✅
- `gofmt` applied repo-wide; `go vet`, `go test`, `go build`, `npm ci`, `npm run build` re-run green post-formatting.
- All 15 acceptance criteria verified individually with evidence ([13](13-acceptance-checklist.md)).
- Documentation set completed; git commit/push intentionally not performed (awaiting authorization).

## Deliberately out of scope (per assignment boundaries)
- Multi-workspace membership per user; email verification/password reset; refresh tokens; rate limiting; containerization.
