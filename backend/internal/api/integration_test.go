package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"team-task-management/backend/internal/config"
	"team-task-management/backend/internal/store"
)

// defaultTestDBURL is the local development cluster; override with
// TEST_DATABASE_URL. Tests skip when the database is not reachable so
// `go test ./...` still passes on machines without PostgreSQL.
const defaultTestDBURL = "postgres://taskapp:change-this-password@localhost:54329/task_management?sslmode=disable"

func newIntegrationServer(t *testing.T) (*httptest.Server, *store.DB) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = defaultTestDBURL
	}
	ctx := context.Background()
	d, err := store.Open(ctx, url)
	if err != nil {
		t.Skipf("integration database not reachable (%v); set TEST_DATABASE_URL to enable", err)
	}
	t.Cleanup(d.Close)
	if err := store.Migrate(ctx, d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := d.Pool.Exec(ctx, `TRUNCATE users, workspaces, workspace_members, projects, tasks RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset database: %v", err)
	}
	cfg := config.Config{Port: "0", DatabaseURL: url, JWTSecret: strings.Repeat("integration-test-secret-", 3), FrontendOrigin: "http://localhost:5173", JWTTTLHours: 1}
	ts := httptest.NewServer(New(d, cfg))
	t.Cleanup(ts.Close)
	return ts, d
}
func doRaw(t *testing.T, ts *httptest.Server, method, path, token string, body any) (int, any) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.URL+path, rd)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}
func doReq(t *testing.T, ts *httptest.Server, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	code, v := doRaw(t, ts, method, path, token, body)
	m, _ := v.(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	return code, m
}
func doReqList(t *testing.T, ts *httptest.Server, method, path, token string, body any) (int, []any) {
	t.Helper()
	code, v := doRaw(t, ts, method, path, token, body)
	l, _ := v.([]any)
	return code, l
}
func num(t *testing.T, m map[string]any, key string) float64 {
	t.Helper()
	v, ok := m[key].(float64)
	if !ok {
		t.Fatalf("key %q missing or not a number in %v", key, m)
	}
	return v
}
func dOffset(days int) string { return time.Now().AddDate(0, 0, days).Format("2006-01-02") }
func istr(n int64) string     { return strconv.FormatInt(n, 10) }
func cloneMap(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// TestIntegrationWorkflow walks the full assignment flow in order; subtests
// share state because later steps depend on earlier ones.
func TestIntegrationWorkflow(t *testing.T) {
	ts, d := newIntegrationServer(t)
	ctx := context.Background()
	rangeStart := time.Date(time.Now().Year()-1, 1, 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	rangeEndTime := time.Date(time.Now().Year()+1, 12, 31, 0, 0, 0, 0, time.Local)
	rangeEnd := rangeEndTime.Format("2006-01-02")
	pastRangeEnd := rangeEndTime.AddDate(0, 0, 1).Format("2006-01-02") // guaranteed outside the project range

	var adminToken, bobToken, caraToken string
	var adminID, bobID, caraID, workspaceID, projectID float64
	var tSpec, tDesign, tShip, tChore, tBug float64

	t.Run("first registered user becomes administrator", func(t *testing.T) {
		code, out := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": "Ada Admin", "email": "admin@example.com", "password": "password123", "confirmPassword": "password123"})
		if code != 201 {
			t.Fatalf("register: %d %v", code, out)
		}
		adminToken, _ = out["token"].(string)
		if adminToken == "" {
			t.Fatal("first user should receive a token")
		}
		u := out["user"].(map[string]any)
		if u["role"] != "Administrator" {
			t.Fatalf("first user role = %v", u["role"])
		}
		adminID, workspaceID = u["id"].(float64), u["workspaceId"].(float64)
		if workspaceID <= 0 {
			t.Fatal("workspace should be created for the first user")
		}
	})
	t.Run("registration validation", func(t *testing.T) {
		if code, _ := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": "Ada Admin", "email": "admin@example.com", "password": "password123", "confirmPassword": "password123"}); code != 409 {
			t.Fatalf("duplicate email: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": "Bad", "email": "bad@example.com", "password": "password123", "confirmPassword": "different123"}); code != 400 {
			t.Fatalf("password mismatch: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": "Bad", "email": "not-an-email", "password": "password123", "confirmPassword": "password123"}); code != 400 {
			t.Fatalf("bad email: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": "Bad", "email": "short@example.com", "password": "short", "confirmPassword": "short"}); code != 400 {
			t.Fatalf("short password: %d", code)
		}
	})
	t.Run("later user registers without workspace", func(t *testing.T) {
		code, out := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": "Bob Member", "email": "bob@example.com", "password": "password123", "confirmPassword": "password123"})
		if code != 201 {
			t.Fatalf("register bob: %d %v", code, out)
		}
		bobID = num(t, out, "userId")
		if tok, _ := out["token"].(string); tok != "" {
			t.Fatal("non-first user must not receive a token")
		}
		if code, _ := doReq(t, ts, "POST", "/api/auth/login", "", map[string]any{"email": "bob@example.com", "password": "password123"}); code != 403 {
			t.Fatalf("unassigned login should be 403, got %d", code)
		}
	})
	t.Run("protected endpoints reject bad authentication", func(t *testing.T) {
		if code, _ := doReq(t, ts, "GET", "/api/projects", "", nil); code != 401 {
			t.Fatalf("no token: %d", code)
		}
		if code, _ := doReq(t, ts, "GET", "/api/projects", "garbage.token.here", nil); code != 401 {
			t.Fatalf("garbage token: %d", code)
		}
	})
	t.Run("admin login and me", func(t *testing.T) {
		code, out := doReq(t, ts, "POST", "/api/auth/login", "", map[string]any{"email": "ADMIN@Example.com", "password": "password123"})
		if code != 200 {
			t.Fatalf("admin login (case-insensitive email): %d %v", code, out)
		}
		adminToken, _ = out["token"].(string)
		code, out = doReq(t, ts, "GET", "/api/me", adminToken, nil)
		if code != 200 || out["role"] != "Administrator" {
			t.Fatalf("me: %d %v", code, out)
		}
		if code, _ := doReq(t, ts, "POST", "/api/auth/login", "", map[string]any{"email": "admin@example.com", "password": "wrong-password"}); code != 401 {
			t.Fatalf("wrong password should be 401, got %d", code)
		}
	})
	t.Run("admin adds member by exact email", func(t *testing.T) {
		if code, _ := doReq(t, ts, "POST", "/api/members", adminToken, map[string]any{"email": "nobody@example.com"}); code != 404 {
			t.Fatalf("unknown email: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/members", adminToken, map[string]any{"email": "bob@example.com"}); code != 201 {
			t.Fatalf("add bob: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/members", adminToken, map[string]any{"email": "  BOB@Example.com  "}); code != 409 {
			t.Fatalf("adding existing member should be 409, got %d", code)
		}
		code, out := doReqList(t, ts, "GET", "/api/members", adminToken, nil)
		if code != 200 || len(out) != 2 {
			t.Fatalf("members list: %d %v", code, out)
		}
	})
	t.Run("member logs in with member role", func(t *testing.T) {
		code, out := doReq(t, ts, "POST", "/api/auth/login", "", map[string]any{"email": "bob@example.com", "password": "password123"})
		if code != 200 {
			t.Fatalf("bob login: %d %v", code, out)
		}
		if out["user"].(map[string]any)["role"] != "Member" {
			t.Fatalf("bob role = %v", out["user"].(map[string]any)["role"])
		}
		bobToken, _ = out["token"].(string)
	})
	t.Run("member cannot manage projects or members", func(t *testing.T) {
		if code, _ := doReq(t, ts, "POST", "/api/projects", bobToken, map[string]any{"name": "Sneaky", "startDate": rangeStart, "endDate": rangeEnd}); code != 403 {
			t.Fatalf("member create project: %d", code)
		}
		if code, _ := doReq(t, ts, "DELETE", "/api/projects/"+istr(int64(projectID)), bobToken, nil); code != 403 {
			t.Fatalf("member delete project: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/members", bobToken, map[string]any{"email": "cara@example.com"}); code != 403 {
			t.Fatalf("member add member: %d", code)
		}
		if code, _ := doReq(t, ts, "DELETE", "/api/members/"+istr(int64(adminID)), bobToken, nil); code != 403 {
			t.Fatalf("member remove member: %d", code)
		}
	})
	t.Run("project validation and creation", func(t *testing.T) {
		if code, _ := doReq(t, ts, "POST", "/api/projects", adminToken, map[string]any{"name": "A", "startDate": rangeStart, "endDate": rangeEnd}); code != 400 {
			t.Fatalf("short name: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/projects", adminToken, map[string]any{"name": "Apollo", "startDate": rangeEnd, "endDate": rangeStart}); code != 400 {
			t.Fatalf("end before start: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/projects", adminToken, map[string]any{"name": "Apollo", "startDate": "2026/01/01", "endDate": rangeEnd}); code != 400 {
			t.Fatalf("bad date format: %d", code)
		}
		code, out := doReq(t, ts, "POST", "/api/projects", adminToken, map[string]any{"name": "Apollo", "description": "Flagship", "startDate": rangeStart, "endDate": rangeEnd})
		if code != 201 {
			t.Fatalf("create project: %d %v", code, out)
		}
		projectID = num(t, out, "id")
		code, list := doReqList(t, ts, "GET", "/api/projects", bobToken, nil)
		if code != 200 || len(list) != 1 {
			t.Fatalf("member read projects: %d %v", code, list)
		}
		if code, _ := doReq(t, ts, "PUT", "/api/projects/"+istr(int64(projectID)), adminToken, map[string]any{"name": "Apollo", "startDate": rangeEnd, "endDate": rangeStart}); code != 400 {
			t.Fatalf("update end before start: %d", code)
		}
	})
	t.Run("second workspace is isolated", func(t *testing.T) {
		code, out := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": "Cara Solo", "email": "cara@example.com", "password": "password123", "confirmPassword": "password123"})
		if code != 201 {
			t.Fatalf("register cara: %d %v", code, out)
		}
		caraID = num(t, out, "userId")
		var caraWS int64
		if err := d.Pool.QueryRow(ctx, `INSERT INTO workspaces(name,admin_user_id) VALUES('Cara Workspace',$1) RETURNING id`, int64(caraID)).Scan(&caraWS); err != nil {
			t.Fatal(err)
		}
		if _, err := d.Pool.Exec(ctx, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'Administrator')`, caraWS, int64(caraID)); err != nil {
			t.Fatal(err)
		}
		code, out = doReq(t, ts, "POST", "/api/auth/login", "", map[string]any{"email": "cara@example.com", "password": "password123"})
		if code != 200 {
			t.Fatalf("cara login: %d %v", code, out)
		}
		caraToken, _ = out["token"].(string)
		if code, _ := doReq(t, ts, "GET", "/api/projects/"+istr(int64(projectID)), caraToken, nil); code != 404 {
			t.Fatalf("cross-workspace project read should be 404, got %d", code)
		}
		code, caraTasks := doReqList(t, ts, "GET", "/api/tasks", caraToken, nil)
		if code != 200 || len(caraTasks) != 0 {
			t.Fatalf("cross-workspace tasks: %d %v", code, caraTasks)
		}
	})
	t.Run("task validation", func(t *testing.T) {
		base := map[string]any{"projectId": projectID, "title": "Write spec", "assignedUserId": bobID, "priority": "High", "dueDate": pastRangeEnd, "status": "To Do"}
		if code, _ := doReq(t, ts, "POST", "/api/tasks", adminToken, base); code != 400 {
			t.Fatalf("due date outside project range: %d", code)
		}
		other := cloneMap(base)
		other["priority"] = "Urgent"
		other["dueDate"] = dOffset(10)
		if code, _ := doReq(t, ts, "POST", "/api/tasks", adminToken, other); code != 400 {
			t.Fatalf("bad priority: %d", code)
		}
		outsider := cloneMap(base)
		outsider["assignedUserId"] = caraID
		outsider["dueDate"] = dOffset(10)
		if code, _ := doReq(t, ts, "POST", "/api/tasks", adminToken, outsider); code != 400 {
			t.Fatalf("assignee outside workspace: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/tasks", bobToken, base); code != 403 {
			t.Fatalf("member create task: %d", code)
		}
	})
	t.Run("admin creates tasks and member sees only own", func(t *testing.T) {
		mk := func(title string, assignee float64, priority, due, status string) float64 {
			code, out := doReq(t, ts, "POST", "/api/tasks", adminToken, map[string]any{"projectId": projectID, "title": title, "description": "", "assignedUserId": assignee, "priority": priority, "dueDate": due, "status": status})
			if code != 201 {
				t.Fatalf("create %s: %d %v", title, code, out)
			}
			return num(t, out, "id")
		}
		tSpec = mk("Write spec", bobID, "High", dOffset(-60), "To Do")
		tDesign = mk("Design API", bobID, "Medium", dOffset(30), "In Progress")
		tShip = mk("Ship release", bobID, "Low", dOffset(45), "Completed")
		tChore = mk("Admin chore", adminID, "Low", dOffset(90), "To Do")
		tBug = mk("Fix bug", bobID, "High", dOffset(-15), "To Do")
		code, out := doReqList(t, ts, "GET", "/api/tasks", adminToken, nil)
		if code != 200 || len(out) != 5 {
			t.Fatalf("admin tasks: %d %v", code, out)
		}
		code, out = doReqList(t, ts, "GET", "/api/tasks", bobToken, nil)
		if code != 200 || len(out) != 4 {
			t.Fatalf("member should see only own 4 tasks, got %d %v", code, out)
		}
		var overdueBug, doneShip int
		if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE id=$1 AND status='To Do' AND due_date<CURRENT_DATE`, int64(tBug)).Scan(&overdueBug); err != nil {
			t.Fatal(err)
		}
		if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE id=$1 AND status='Completed'`, int64(tShip)).Scan(&doneShip); err != nil {
			t.Fatal(err)
		}
		if overdueBug != 1 || doneShip != 1 {
			t.Fatalf("statuses not persisted as created: overdueBug=%d doneShip=%d", overdueBug, doneShip)
		}
	})
	t.Run("task filters", func(t *testing.T) {
		assertCount := func(query string, want int) {
			t.Helper()
			code, out := doReqList(t, ts, "GET", "/api/tasks"+query, adminToken, nil)
			if code != 200 {
				t.Fatalf("filter %s: %d", query, code)
			}
			if got := len(out); got != want {
				t.Fatalf("filter %s: got %d tasks, want %d", query, got, want)
			}
		}
		assertCount("?status=Completed", 1)
		assertCount("?priority=High", 2)
		assertCount("?projectId="+istr(int64(projectID)), 5)
		assertCount("?status=To%20Do&priority=Low", 1)
		assertCount("?status=In%20Progress&priority=High", 0)
		if code, _ := doReq(t, ts, "GET", "/api/tasks?priority=Nope", adminToken, nil); code != 400 {
			t.Fatalf("invalid filter should be 400, got %d", code)
		}
	})
	t.Run("member task permissions", func(t *testing.T) {
		if code, _ := doReq(t, ts, "PUT", "/api/tasks/"+istr(int64(tChore)), bobToken, map[string]any{"status": "Completed"}); code != 403 {
			t.Fatalf("member update another's task: %d", code)
		}
		full := map[string]any{"projectId": projectID, "title": "Hijack", "description": "", "assignedUserId": bobID, "priority": "High", "dueDate": dOffset(3), "status": "To Do"}
		if code, _ := doReq(t, ts, "PUT", "/api/tasks/"+istr(int64(tSpec)), bobToken, full); code != 403 {
			t.Fatalf("member full edit: %d", code)
		}
		if code, _ := doReq(t, ts, "PUT", "/api/tasks/"+istr(int64(tSpec)), bobToken, map[string]any{"status": "Completed"}); code != 200 {
			t.Fatalf("member status update own task: %d", code)
		}
		var status string
		if err := d.Pool.QueryRow(ctx, `SELECT status FROM tasks WHERE id=$1`, int64(tSpec)).Scan(&status); err != nil || status != "Completed" {
			t.Fatalf("status not persisted: %v %q", err, status)
		}
		if code, _ := doReq(t, ts, "PUT", "/api/tasks/"+istr(int64(tSpec)), bobToken, map[string]any{"status": "Done"}); code != 400 {
			t.Fatalf("invalid status: %d", code)
		}
	})
	t.Run("admin status-only update works", func(t *testing.T) {
		if code, _ := doReq(t, ts, "PUT", "/api/tasks/"+istr(int64(tChore)), adminToken, map[string]any{"status": "In Progress"}); code != 200 {
			t.Fatalf("admin status-only update: %d", code)
		}
	})
	t.Run("dashboard counts", func(t *testing.T) {
		code, out := doReq(t, ts, "GET", "/api/dashboard", adminToken, nil)
		if code != 200 {
			t.Fatalf("admin dashboard: %d", code)
		}
		for key, want := range map[string]float64{"totalProjects": 1, "totalTasks": 5, "completedTasks": 2, "pendingTasks": 3, "overdueTasks": 1} {
			if got := num(t, out, key); got != want {
				t.Fatalf("admin %s = %v, want %v (%v)", key, got, want, out)
			}
		}
		code, out = doReq(t, ts, "GET", "/api/dashboard", bobToken, nil)
		if code != 200 {
			t.Fatalf("member dashboard: %d", code)
		}
		for key, want := range map[string]float64{"totalProjects": 1, "totalTasks": 4, "completedTasks": 2, "pendingTasks": 2, "overdueTasks": 1} {
			if got := num(t, out, key); got != want {
				t.Fatalf("member %s = %v, want %v (%v)", key, got, want, out)
			}
		}
	})
	t.Run("admin full edit and reassign", func(t *testing.T) {
		full := map[string]any{"projectId": projectID, "title": "Admin chore renamed", "description": "updated", "assignedUserId": adminID, "priority": "Medium", "dueDate": dOffset(90), "status": "To Do"}
		if code, _ := doReq(t, ts, "PUT", "/api/tasks/"+istr(int64(tChore)), adminToken, full); code != 200 {
			t.Fatalf("admin full edit: %d", code)
		}
		bad := cloneMap(full)
		bad["dueDate"] = pastRangeEnd
		if code, _ := doReq(t, ts, "PUT", "/api/tasks/"+istr(int64(tChore)), adminToken, bad); code != 400 {
			t.Fatalf("edit due outside range: %d", code)
		}
		bad2 := cloneMap(full)
		bad2["assignedUserId"] = caraID
		if code, _ := doReq(t, ts, "PUT", "/api/tasks/"+istr(int64(tChore)), adminToken, bad2); code != 400 {
			t.Fatalf("reassign outside workspace: %d", code)
		}
	})
	t.Run("project update cannot strand task due dates", func(t *testing.T) {
		shrunk := map[string]any{"name": "Apollo", "description": "Flagship", "startDate": dOffset(-7), "endDate": dOffset(300)}
		if code, out := doReq(t, ts, "PUT", "/api/projects/"+istr(int64(projectID)), adminToken, shrunk); code != 409 {
			t.Fatalf("expected 409 for stranded due dates, got %d %v", code, out)
		}
		widened := map[string]any{"name": "Apollo Renamed", "description": "Flagship", "startDate": rangeStart, "endDate": rangeEnd}
		if code, _ := doReq(t, ts, "PUT", "/api/projects/"+istr(int64(projectID)), adminToken, widened); code != 200 {
			t.Fatalf("valid update: %d", code)
		}
	})
	t.Run("task delete is admin only", func(t *testing.T) {
		if code, _ := doReq(t, ts, "DELETE", "/api/tasks/"+istr(int64(tDesign)), bobToken, nil); code != 403 {
			t.Fatalf("member delete task: %d", code)
		}
		if code, _ := doReq(t, ts, "DELETE", "/api/tasks/"+istr(int64(tDesign)), adminToken, nil); code != 200 {
			t.Fatalf("admin delete task: %d", code)
		}
	})
	t.Run("removing member revokes access and reassigns tasks", func(t *testing.T) {
		if code, _ := doReq(t, ts, "DELETE", "/api/members/"+istr(int64(adminID)), adminToken, nil); code != 400 {
			t.Fatalf("admin self-removal should be 400, got %d", code)
		}
		if code, _ := doReq(t, ts, "DELETE", "/api/members/"+istr(int64(bobID)), adminToken, nil); code != 200 {
			t.Fatalf("remove bob: %d", code)
		}
		var bobTasks, adminTasks int
		if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE assigned_user_id=$1`, int64(bobID)).Scan(&bobTasks); err != nil {
			t.Fatal(err)
		}
		if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE assigned_user_id=$1`, int64(adminID)).Scan(&adminTasks); err != nil {
			t.Fatal(err)
		}
		if bobTasks != 0 || adminTasks != 4 {
			t.Fatalf("tasks not reassigned: bob=%d admin=%d", bobTasks, adminTasks)
		}
		if code, _ := doReq(t, ts, "GET", "/api/projects", bobToken, nil); code != 403 {
			t.Fatalf("removed member should lose access, got %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/auth/login", "", map[string]any{"email": "bob@example.com", "password": "password123"}); code != 403 {
			t.Fatalf("removed member login should be 403, got %d", code)
		}
		_, members := doReqList(t, ts, "GET", "/api/members", adminToken, nil)
		if len(members) != 1 {
			t.Fatalf("members after removal: %v", members)
		}
		code, out := doReq(t, ts, "GET", "/api/dashboard", adminToken, nil)
		if code != 200 || num(t, out, "totalTasks") != 4 {
			t.Fatalf("dashboard after removal: %d %v", code, out)
		}
	})
	t.Run("project delete cascades tasks", func(t *testing.T) {
		code, out := doReq(t, ts, "POST", "/api/projects", adminToken, map[string]any{"name": "Temp", "startDate": dOffset(-1), "endDate": dOffset(1)})
		if code != 201 {
			t.Fatalf("temp project: %d %v", code, out)
		}
		pid := int64(num(t, out, "id"))
		code, out = doReq(t, ts, "POST", "/api/tasks", adminToken, map[string]any{"projectId": pid, "title": "Temp task", "assignedUserId": adminID, "priority": "Low", "dueDate": dOffset(0), "status": "To Do"})
		if code != 201 {
			t.Fatalf("temp task: %d %v", code, out)
		}
		tid := int64(num(t, out, "id"))
		if code, _ := doReq(t, ts, "DELETE", "/api/projects/"+istr(pid), adminToken, nil); code != 200 {
			t.Fatalf("delete project: %d", code)
		}
		var n int
		if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE id=$1`, tid).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatal("orphaned task survived project deletion")
		}
	})
}
