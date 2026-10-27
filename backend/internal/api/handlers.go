package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"team-task-management/backend/internal/store"
)

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	i := current(r)
	var projects, tasks, completed, pending, overdue int
	var err error
	if i.Role == "Administrator" {
		err = s.db.Pool.QueryRow(r.Context(), `SELECT count(*) FROM projects WHERE workspace_id=$1`, i.WorkspaceID).Scan(&projects)
	} else {
		err = s.db.Pool.QueryRow(r.Context(), `SELECT count(DISTINCT p.id) FROM projects p JOIN tasks t ON t.project_id=p.id WHERE p.workspace_id=$1 AND t.assigned_user_id=$2`, i.WorkspaceID, i.UserID).Scan(&projects)
	}
	if err != nil {
		fail(w, 500, "Could not load dashboard")
		return
	}
	base := ` FROM tasks t JOIN projects p ON p.id=t.project_id WHERE p.workspace_id=$1`
	args := []any{i.WorkspaceID}
	if i.Role != "Administrator" {
		base += ` AND t.assigned_user_id=$2`
		args = append(args, i.UserID)
	}
	if err = s.db.Pool.QueryRow(r.Context(), `SELECT count(*)`+base, args...).Scan(&tasks); err != nil {
		fail(w, 500, "Could not load dashboard")
		return
	}
	if err = s.db.Pool.QueryRow(r.Context(), `SELECT count(*)`+base+` AND t.status='Completed'`, args...).Scan(&completed); err != nil {
		fail(w, 500, "Could not load dashboard")
		return
	}
	if err = s.db.Pool.QueryRow(r.Context(), `SELECT count(*)`+base+` AND t.status<>'Completed'`, args...).Scan(&pending); err != nil {
		fail(w, 500, "Could not load dashboard")
		return
	}
	if err = s.db.Pool.QueryRow(r.Context(), `SELECT count(*)`+base+` AND t.status<>'Completed' AND t.due_date<CURRENT_DATE`, args...).Scan(&overdue); err != nil {
		fail(w, 500, "Could not load dashboard")
		return
	}
	writeJSON(w, 200, map[string]int{"totalProjects": projects, "totalTasks": tasks, "completedTasks": completed, "pendingTasks": pending, "overdueTasks": overdue})
}
func (s *Server) members(w http.ResponseWriter, r *http.Request) {
	i := current(r)
	rows, err := s.db.Pool.Query(r.Context(), `SELECT u.id,u.name,u.email,wm.role FROM workspace_members wm JOIN users u ON u.id=wm.user_id WHERE wm.workspace_id=$1 ORDER BY u.name`, i.WorkspaceID)
	if err != nil {
		fail(w, 500, "Could not load members")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, email, role string
		if rows.Scan(&id, &name, &email, &role) == nil {
			out = append(out, map[string]any{"id": id, "name": name, "email": email, "role": role})
		}
	}
	writeJSON(w, 200, out)
}
func (s *Server) addMember(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if decode(r, &in) != nil {
		fail(w, 400, "Invalid request")
		return
	}
	in.Email = store.NormalizeEmail(in.Email)
	if !emailPattern.MatchString(in.Email) {
		fail(w, 400, "Enter a valid registered email address")
		return
	}
	i := current(r)
	var uid int64
	err := s.db.Pool.QueryRow(r.Context(), `SELECT id FROM users WHERE email=$1`, in.Email).Scan(&uid)
	if err != nil {
		fail(w, 404, "No registered user found with that exact email")
		return
	}
	var existing int64
	err = s.db.Pool.QueryRow(r.Context(), `SELECT workspace_id FROM workspace_members WHERE user_id=$1`, uid).Scan(&existing)
	if err == nil {
		if existing == i.WorkspaceID {
			fail(w, 409, "That user is already a member of this workspace")
			return
		}
		fail(w, 409, "That user already belongs to another workspace")
		return
	}
	if !store.IsNoRows(err) {
		fail(w, 500, "Could not add member")
		return
	}
	if _, err = s.db.Pool.Exec(r.Context(), `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'Member')`, i.WorkspaceID, uid); err != nil {
		fail(w, 500, "Could not add member")
		return
	}
	writeJSON(w, 201, map[string]string{"message": "Team member added"})
}
func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "Invalid member ID")
		return
	}
	i := current(r)
	if id == i.UserID {
		fail(w, 400, "Administrator cannot remove their own account")
		return
	}
	// Remove the membership and move the member's tasks to the administrator in
	// one transaction so no task is ever left assigned to a non-member.
	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		fail(w, 500, "Could not remove member")
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND role='Member'`, i.WorkspaceID, id)
	if err != nil {
		fail(w, 500, "Could not remove member")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "Member not found")
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE tasks t SET assigned_user_id=$1,updated_at=now() FROM projects p WHERE t.project_id=p.id AND p.workspace_id=$2 AND t.assigned_user_id=$3`, i.UserID, i.WorkspaceID, id); err != nil {
		fail(w, 500, "Could not reassign removed member's tasks")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 500, "Could not remove member")
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Member removed; their tasks were reassigned to you"})
}

type projectInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
}

func validDate(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }
func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
	i := current(r)
	rows, err := s.db.Pool.Query(r.Context(), `SELECT id,name,description,start_date::text,end_date::text FROM projects WHERE workspace_id=$1 ORDER BY start_date DESC,id DESC`, i.WorkspaceID)
	if err != nil {
		fail(w, 500, "Could not load projects")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var n, d, sd, ed string
		if rows.Scan(&id, &n, &d, &sd, &ed) == nil {
			out = append(out, map[string]any{"id": id, "name": n, "description": d, "startDate": sd, "endDate": ed})
		}
	}
	writeJSON(w, 200, out)
}
func validProject(in projectInput) (projectInput, bool) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	a, e1 := validDate(in.StartDate)
	b, e2 := validDate(in.EndDate)
	if len(in.Name) < 2 || len(in.Name) > 160 || len(in.Description) > 2000 || e1 != nil || e2 != nil || b.Before(a) {
		return in, false
	}
	return in, true
}
func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var in projectInput
	if decode(r, &in) != nil {
		fail(w, 400, "Invalid request")
		return
	}
	in, ok := validProject(in)
	if !ok {
		fail(w, 400, "Project name is required (2-160 characters) and dates must be valid with end date not before start date")
		return
	}
	i := current(r)
	var id int64
	err := s.db.Pool.QueryRow(r.Context(), `INSERT INTO projects(workspace_id,name,description,start_date,end_date) VALUES($1,$2,$3,$4,$5) RETURNING id`, i.WorkspaceID, in.Name, in.Description, in.StartDate, in.EndDate).Scan(&id)
	if err != nil {
		fail(w, 400, "Could not create project")
		return
	}
	writeJSON(w, 201, map[string]any{"id": id, "message": "Project created"})
}
func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "Invalid project ID")
		return
	}
	i := current(r)
	var n, d, sd, ed string
	err = s.db.Pool.QueryRow(r.Context(), `SELECT name,description,start_date::text,end_date::text FROM projects WHERE id=$1 AND workspace_id=$2`, id, i.WorkspaceID).Scan(&n, &d, &sd, &ed)
	if err != nil {
		fail(w, 404, "Project not found")
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "name": n, "description": d, "startDate": sd, "endDate": ed})
}
func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "Invalid project ID")
		return
	}
	var in projectInput
	if decode(r, &in) != nil {
		fail(w, 400, "Invalid request")
		return
	}
	in, ok := validProject(in)
	if !ok {
		fail(w, 400, "Project name is required and dates must be valid with end date not before start date")
		return
	}
	i := current(r)
	// Preserve the invariant that every task due date stays inside the project
	// range: reject a reschedule that would strand existing tasks outside it.
	var outside int
	if err = s.db.Pool.QueryRow(r.Context(), `SELECT count(*) FROM tasks WHERE project_id=$1 AND (due_date<$2 OR due_date>$3)`, id, in.StartDate, in.EndDate).Scan(&outside); err == nil && outside > 0 {
		fail(w, 409, "Existing task due dates fall outside the new project date range; reschedule those tasks first")
		return
	}
	tag, err := s.db.Pool.Exec(r.Context(), `UPDATE projects SET name=$1,description=$2,start_date=$3,end_date=$4 WHERE id=$5 AND workspace_id=$6`, in.Name, in.Description, in.StartDate, in.EndDate, id, i.WorkspaceID)
	if err != nil {
		fail(w, 400, "Project update failed; check existing task due dates")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "Project not found")
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Project updated"})
}
func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "Invalid project ID")
		return
	}
	i := current(r)
	tag, err := s.db.Pool.Exec(r.Context(), `DELETE FROM projects WHERE id=$1 AND workspace_id=$2`, id, i.WorkspaceID)
	if err != nil {
		fail(w, 500, "Project deletion failed")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "Project not found")
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Project and its tasks deleted"})
}

type taskInput struct {
	ProjectID      int64  `json:"projectId"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	AssignedUserID int64  `json:"assignedUserId"`
	Priority       string `json:"priority"`
	DueDate        string `json:"dueDate"`
	Status         string `json:"status"`
}

// taskUpdate uses pointers so a request that carries only {"status":...} — the
// shape both the admin table dropdown and member status updates send — can be
// distinguished from a full administrative edit.
type taskUpdate struct {
	ProjectID      *int64  `json:"projectId"`
	Title          *string `json:"title"`
	Description    *string `json:"description"`
	AssignedUserID *int64  `json:"assignedUserId"`
	Priority       *string `json:"priority"`
	DueDate        *string `json:"dueDate"`
	Status         *string `json:"status"`
}

func (s *Server) tasks(w http.ResponseWriter, r *http.Request) {
	i := current(r)
	q := `SELECT t.id,t.project_id,p.name,t.title,t.description,t.assigned_user_id,u.name,t.priority,t.due_date::text,t.status FROM tasks t JOIN projects p ON p.id=t.project_id JOIN users u ON u.id=t.assigned_user_id JOIN workspace_members wm ON wm.user_id=u.id AND wm.workspace_id=p.workspace_id WHERE p.workspace_id=$1`
	args := []any{i.WorkspaceID}
	n := 2
	if i.Role != "Administrator" {
		q += ` AND t.assigned_user_id=$2`
		args = append(args, i.UserID)
		n++
	}
	for _, f := range []struct{ key, col string }{{"projectId", "t.project_id"}, {"status", "t.status"}, {"priority", "t.priority"}} {
		if v := r.URL.Query().Get(f.key); v != "" {
			if f.key == "projectId" {
				if _, err := strconv.ParseInt(v, 10, 64); err != nil {
					fail(w, 400, "projectId filter must be numeric")
					return
				}
			}
			if f.key == "priority" && v != "Low" && v != "Medium" && v != "High" {
				fail(w, 400, "priority filter must be Low, Medium, or High")
				return
			}
			if f.key == "status" && v != "To Do" && v != "In Progress" && v != "Completed" {
				fail(w, 400, "status filter must be To Do, In Progress, or Completed")
				return
			}
			q += ` AND ` + f.col + `=$` + strconv.Itoa(n)
			args = append(args, v)
			n++
		}
	}
	q += ` ORDER BY t.due_date ASC,t.id DESC`
	rows, err := s.db.Pool.Query(r.Context(), q, args...)
	if err != nil {
		fail(w, 500, "Could not load tasks")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, pid, uid int64
		var pn, title, desc, an, pri, due, status string
		if rows.Scan(&id, &pid, &pn, &title, &desc, &uid, &an, &pri, &due, &status) == nil {
			out = append(out, map[string]any{"id": id, "projectId": pid, "projectName": pn, "title": title, "description": desc, "assignedUserId": uid, "assigneeName": an, "priority": pri, "dueDate": due, "status": status})
		}
	}
	writeJSON(w, 200, out)
}
func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var in taskInput
	if decode(r, &in) != nil {
		fail(w, 400, "Invalid request")
		return
	}
	if err := validateTask(in); err != nil {
		fail(w, 400, err.Error())
		return
	}
	i := current(r)
	var id int64
	err := s.db.Pool.QueryRow(r.Context(), `INSERT INTO tasks(project_id,title,description,assigned_user_id,priority,due_date,status) SELECT p.id,$2,$3,$4,$5,$6,$7 FROM projects p JOIN workspace_members wm ON wm.workspace_id=p.workspace_id AND wm.user_id=$4 WHERE p.id=$1 AND p.workspace_id=$8 AND $6 BETWEEN p.start_date AND p.end_date RETURNING id`, in.ProjectID, strings.TrimSpace(in.Title), strings.TrimSpace(in.Description), in.AssignedUserID, in.Priority, in.DueDate, in.Status, i.WorkspaceID).Scan(&id)
	if err != nil {
		fail(w, 400, "Task could not be created: the project and assignee must belong to your workspace and the due date must fall within the project date range")
		return
	}
	writeJSON(w, 201, map[string]any{"id": id, "message": "Task created"})
}
func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "Invalid task ID")
		return
	}
	var in taskUpdate
	if decode(r, &in) != nil {
		fail(w, 400, "Invalid request")
		return
	}
	i := current(r)
	var owner, workspace int64
	err = s.db.Pool.QueryRow(r.Context(), `SELECT t.assigned_user_id,p.workspace_id FROM tasks t JOIN projects p ON p.id=t.project_id WHERE t.id=$1`, id).Scan(&owner, &workspace)
	if err != nil || workspace != i.WorkspaceID {
		fail(w, 404, "Task not found")
		return
	}
	statusOnly := in.Status != nil && in.ProjectID == nil && in.Title == nil && in.Description == nil && in.AssignedUserID == nil && in.Priority == nil && in.DueDate == nil
	if statusOnly {
		status := *in.Status
		if status != "To Do" && status != "In Progress" && status != "Completed" {
			fail(w, 400, "Status must be To Do, In Progress, or Completed")
			return
		}
		if owner != i.UserID && i.Role != "Administrator" {
			fail(w, 403, "Only the assignee or an administrator can update a task's status")
			return
		}
		var tag interface{ RowsAffected() int64 }
		if i.Role == "Administrator" {
			tag, err = s.db.Pool.Exec(r.Context(), `UPDATE tasks t SET status=$1,updated_at=now() FROM projects p WHERE t.id=$2 AND t.project_id=p.id AND p.workspace_id=$3`, status, id, i.WorkspaceID)
		} else {
			tag, err = s.db.Pool.Exec(r.Context(), `UPDATE tasks SET status=$1,updated_at=now() WHERE id=$2 AND assigned_user_id=$3`, status, id, i.UserID)
		}
		if err != nil {
			fail(w, 400, "Could not update task status")
			return
		}
		if tag.RowsAffected() == 0 {
			fail(w, 404, "Task not found")
			return
		}
		writeJSON(w, 200, map[string]string{"message": "Task status updated"})
		return
	}
	if i.Role != "Administrator" {
		fail(w, 403, "Team members may only update the status of tasks assigned to them")
		return
	}
	full := taskInput{ProjectID: deref64(in.ProjectID), Title: deref(in.Title), Description: deref(in.Description), AssignedUserID: deref64(in.AssignedUserID), Priority: deref(in.Priority), DueDate: deref(in.DueDate), Status: deref(in.Status)}
	if err := validateTask(full); err != nil {
		fail(w, 400, err.Error())
		return
	}
	tag, err := s.db.Pool.Exec(r.Context(), `UPDATE tasks t SET project_id=$1,title=$2,description=$3,assigned_user_id=$4,priority=$5,due_date=$6,status=$7,updated_at=now() FROM projects p JOIN workspace_members wm ON wm.workspace_id=p.workspace_id AND wm.user_id=$4 WHERE t.id=$8 AND p.id=$1 AND p.workspace_id=$9 AND $6 BETWEEN p.start_date AND p.end_date`, full.ProjectID, strings.TrimSpace(full.Title), strings.TrimSpace(full.Description), full.AssignedUserID, full.Priority, full.DueDate, full.Status, id, i.WorkspaceID)
	if err != nil {
		fail(w, 400, "Task update failed")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 400, "Task update failed: the project and assignee must belong to your workspace and the due date must fall within the project date range")
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Task updated"})
}
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func deref64(n *int64) int64 {
	if n == nil {
		return 0
	}
	return *n
}
func validateTask(in taskInput) error {
	if in.ProjectID < 1 || in.AssignedUserID < 1 || len(strings.TrimSpace(in.Title)) < 2 || len(in.Title) > 200 || len(in.Description) > 2000 {
		return errText("Project, title (2-200 characters), and assignee are required")
	}
	if in.Priority != "Low" && in.Priority != "Medium" && in.Priority != "High" {
		return errText("Priority must be Low, Medium, or High")
	}
	if in.Status == "" {
		in.Status = "To Do"
	}
	if in.Status != "To Do" && in.Status != "In Progress" && in.Status != "Completed" {
		return errText("Invalid task status")
	}
	if _, err := validDate(in.DueDate); err != nil {
		return errText("Due date must use YYYY-MM-DD format")
	}
	return nil
}

type simpleError string

func (e simpleError) Error() string { return string(e) }
func errText(s string) error        { return simpleError(s) }
func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, 400, "Invalid task ID")
		return
	}
	i := current(r)
	tag, err := s.db.Pool.Exec(r.Context(), `DELETE FROM tasks t USING projects p WHERE t.id=$1 AND t.project_id=p.id AND p.workspace_id=$2`, id, i.WorkspaceID)
	if err != nil {
		fail(w, 500, "Could not delete task")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "Task not found")
		return
	}
	writeJSON(w, 200, map[string]string{"message": "Task deleted"})
}
