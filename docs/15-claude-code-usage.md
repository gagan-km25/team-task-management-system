# 15 — Claude Code Usage

The project was developed with **Claude Code** (CLI agent in VS Code), running on the user's configured **Z.ai GLM Coding Plan** as the model provider. The agent executed every shell command, code edit, test run, and database operation in this repository; the human directed the work via the prompts in [05-prompt-log.md](05-prompt-log.md) and made the two explicit calls noted below.

## How the agent worked
- **Audit before edit** — read the entire starter (backend, frontend, schema, docs, tests) and produced a defect list with a requirement→implementation mapping before changing code.
- **Verification-first discipline** — no completion claim without an executed command: `gofmt`, `go vet`, `go test`, `go build`, `npm ci`, `npm run build`, live-server E2E, direct SQL constraint probes.
- **Incremental fixes preserving the starter** — the existing structure (Go module layout, React app, schema) was kept; defects were patched in place rather than rebuilt.
- **Regression protection** — every fixed defect gained a test (e.g., admin status-only update, member removal reassignment, project reschedule guard).
- **Environment autonomy within bounds** — installed a user-local Go toolchain and portable PostgreSQL (no admin rights, outside the repo) when the required stack was absent; cleared rebuildable caches when the full disk blocked builds; never touched unrelated user data.
- **Safety rules honored** — no secrets printed or committed (none present in the project tree); no git commit or push (not authorized at the time; the repository was initialized and published afterwards by the user); the GLM API key was never exposed or referenced.

## Human decisions during the session
1. Approval was implicit in the initial directives ("proceed without waiting for approval between routine steps"); no mid-course approvals were requested.
2. The GitHub publication step remains intentionally open for the user (no push authorization).

## Practical notes for future sessions
- Go and PostgreSQL are user-local: prepend `$HOME/tools/go/bin` and `$HOME/tools/pgsql/bin` to `PATH` in Git Bash (adjust to your install location).
- The dev cluster runs on port **54329**; integration tests read `TEST_DATABASE_URL` and default to it.
- `pg_ctl start` misbehaves under Git Bash on this machine; start `postgres.exe` directly instead.
- The system drive is nearly full — check `df -h /c` before large builds.
