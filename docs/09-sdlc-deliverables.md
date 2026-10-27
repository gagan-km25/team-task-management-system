# 09 — SDLC Deliverables

Where each software-development-lifecycle artifact lives, and the quality gates it passed.

## Artifact map
| SDLC artifact | Location |
|---|---|
| Problem statement / assignment intake | [01-assignment-overview.md](01-assignment-overview.md) |
| Requirements specification | [02-requirements.md](02-requirements.md) |
| User stories | [03-user-stories.md](03-user-stories.md) |
| Architecture / design | [04-architecture.md](04-architecture.md) |
| Database design | [11-database-design.md](11-database-design.md) + `backend/db/schema.sql`, `backend/internal/store/migrations/` |
| API contract | [10-api-reference.md](10-api-reference.md) |
| Test plan | [06-test-plan.md](06-test-plan.md) |
| Automated tests | `backend/internal/api/unit_test.go`, `backend/internal/api/integration_test.go`, `backend/internal/config/config_test.go`, `e2e.py` (external, 44 checks) |
| Sprint / delivery plan | [07-sprint-plan.md](07-sprint-plan.md) |
| Prompt / decision log | [05-prompt-log.md](05-prompt-log.md), [15-claude-code-usage.md](15-claude-code-usage.md) |
| Execution evidence | [14-execution-commands.md](14-execution-commands.md) |
| Defect tracking | [08-known-issues.md](08-known-issues.md), `CHANGELOG.md` |
| Security review | [12-security-review.md](12-security-review.md) |
| Acceptance verification | [13-acceptance-checklist.md](13-acceptance-checklist.md) |
| Demo / handover | [16-demo-script.md](16-demo-script.md), [README.md](../README.md) |
| Contributor guidance | [CLAUDE.md](../CLAUDE.md) |

## Quality gates (final state, 2026-10-27)
| Gate | Result |
|---|---|
| `gofmt -l .` (backend) | clean |
| `go vet ./...` | pass |
| `go test ./...` | pass (31 tests / subtests across 2 packages) |
| `go test -race ./...` | blocked — environment (see [08](08-known-issues.md)) |
| `go build ./...` | pass |
| `npm ci` | pass (runs from `package-lock.json`, which exists in the working tree; no Git repository yet) |
| `npm run build` | pass |
| Live-server E2E | 44/44 pass |
| DB constraint probes | 8/8 raise expected errors; cascade verified |
| Persistence across restart | verified |
| All 15 acceptance criteria | PASS with evidence ([13](13-acceptance-checklist.md)) |

## Change management
- All changes were made in place on the starter; no rewrite. Each defect fix is paired with a regression test.
- Git: no repository initialized yet and nothing committed or pushed — the publisher step awaits explicit user authorization (`git init`, initial commit, push to their GitHub remote).
