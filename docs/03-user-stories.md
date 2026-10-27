# 03 — User Stories

## Administrator
| ID | Story | Verified by |
|---|---|---|
| A1 | As a first-time visitor I register with my name, email, and password (confirmed) and become the Administrator of my own workspace, signed in immediately. | Integration "first registered user becomes administrator"; e2e "register first user" |
| A2 | As an administrator I log in with my email and password and see my workspace dashboard. | e2e "admin login"; "admin dashboard counts" |
| A3 | As an administrator I add a registered person to my team by entering their exact email, and I see them listed with name, email, and role. | e2e "admin adds member by exact email"; "member list has name/email/role" |
| A4 | As an administrator I remove a team member; they immediately lose access and their tasks return to me so none are lost. | Integration "removing member revokes access and reassigns tasks" |
| A5 | As an administrator I create projects with a name, description, start and end date; I cannot save an end date before the start date. | e2e "project end before start -> 400"; "admin creates project" |
| A6 | As an administrator I edit or delete projects; deleting a project removes its tasks with it. | Integration "project delete cascades tasks" |
| A7 | As an administrator I create tasks inside a project: title, description, assignee, priority, due date, status. The assignee must be on my team and the due date inside the project window. | e2e task validation + creation checks |
| A8 | As an administrator I edit, reassign, or delete any task, and I can flip any task's status from the task table. | Integration "admin full edit and reassign"; "admin status-only update works" |
| A9 | As an administrator my dashboard shows total projects, total tasks, completed, pending, and overdue counts for the whole workspace. | Integration "dashboard counts" |
| A10 | As an administrator I cannot remove my own account. | e2e "admin cannot remove self -> 400" |

## Member
| ID | Story | Verified by |
|---|---|---|
| M1 | As a registered person who is not yet in a workspace, I can create an account but cannot see anyone's projects or tasks until invited. | Integration "later user registers without workspace" |
| M2 | As a member I log in and see a dashboard summarizing only the tasks assigned to me. | Integration "dashboard counts" (member-scoped) |
| M3 | As a member I see only the tasks assigned to me, and I can filter them by project, status, and priority. | Integration "admin creates tasks and member sees only own"; "task filters" |
| M4 | As a member I can update the status of my own tasks but nothing else about them, and never another member's tasks. | Integration "member task permissions" |
| M5 | As a member I cannot create projects, tasks, or modify the team. | e2e "member cannot ..." checks |
| M6 | When I am removed from a workspace, my sessions stop working immediately even if my token has not expired. | Integration "removed member loses access" |
