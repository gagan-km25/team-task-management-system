# 16 — Demo Script

A ~10-minute walkthrough of the running system. Start both processes first (commands in [14-execution-commands.md](14-execution-commands.md) / README): backend on :8080, frontend via `npm run dev` (Vite prints the URL, default http://localhost:5173).

## 1. First user becomes administrator (2 min)
1. Open the app → split-screen auth page ("TeamFlow").
2. Choose **Create an account**, register `Ada Admin` / `ada@example.com` / password + confirmation (show the client-side mismatch error by typing different confirmations first).
3. You land signed-in on the **Overview** dashboard with an empty workspace (empty states visible: no overdue, no projects yet).
4. Sidebar shows "Ada's workspace — Administrator".

## 2. Team management (2 min)
5. In a second browser profile/incognito, register `Bob Member` / `bob@example.com` — note the message: registration succeeded, but ask the administrator to add you (no workspace access; try logging in → "not currently a member of a workspace").
6. Back as Ada, open **Team** → **Add member** → enter `bob@example.com` (show the 404 by first typing an unregistered email). Bob appears with role **Member**.
7. Log in as Bob in the second profile — he now gets in, sees an empty member dashboard.

## 3. Projects and validation (2 min)
8. As Ada, **Projects** → **New project**: "Apollo", description, start `2026-01-01`, end `2026-12-31` (first try end before start to show the frontend `min` attribute + backend 400 message).
9. Show the project card (dates, progress bar, task counters).

## 4. Tasks and permissions (2 min)
10. As Ada, **Create task**: "Write spec" → project Apollo, assignee Bob, priority High, due `2026-11-20`. First try a due date outside the project range → error message; then valid.
11. As Bob, **My Tasks**: sees the task (cannot see Ada's), changes its **status dropdown** to Completed — status-only update works. Try Edit/Delete → controls are not shown to members (and the API would 403).
12. As Ada: same task visible on the board with Bob as assignee; use the admin status dropdown (this exercises the status-only path for admins too).

## 5. Filters and dashboard accuracy (1 min)
13. Create 2–3 more tasks with mixed priorities/statuses, one with a past due date inside the project range.
14. Show filters (project / status / priority) updating the table and the count line.
15. **Overview**: total projects/tasks/completed/pending match the visible data; the overdue panel lists exactly the past-due, not-completed task(s). Log in as Bob → dashboard reflects only his tasks.

## 6. Security highlights (1 min)
16. In a terminal: `curl http://localhost:8080/api/projects` → 401; with Bob's old token after Ada **removes Bob** from the Team page → 403 "Workspace membership is required"; Bob's next login → 403.
17. Point out Bob's tasks moved to Ada on removal (Team page + task board) — nothing orphaned.

## 7. Persistence proof (30 s)
18. Stop the backend (Ctrl+C), restart it, refresh the app — everything is still there (PostgreSQL + startup migrations idempotent).

## Recovery notes for demos
- If the workspace gets messy, reset with: `TRUNCATE users, workspaces, workspace_members, projects, tasks RESTART IDENTITY CASCADE;` (dev database only) and start the walkthrough from step 1.
