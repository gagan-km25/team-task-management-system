package api

import (
	"context"
	"testing"
)

// TestPendingRegistrations covers the admin pending-registrations feature:
// derived list, add-to-team via the membership API, per-workspace dismissal
// (persisted, account untouched), authorization, and workspace isolation.
func TestPendingRegistrations(t *testing.T) {
	ts, d := newIntegrationServer(t)
	ctx := context.Background()

	code, out := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": "Reg Admin", "email": "regadmin@example.com", "password": "password123", "confirmPassword": "password123"})
	if code != 201 {
		t.Fatalf("register admin: %d %v", code, out)
	}
	adminToken, _ := out["token"].(string)
	var adminWS int64
	if err := d.Pool.QueryRow(ctx, `SELECT id FROM workspaces LIMIT 1`).Scan(&adminWS); err != nil {
		t.Fatal(err)
	}

	mkUser := func(name, email string) float64 {
		code, out := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": name, "email": email, "password": "password123", "confirmPassword": "password123"})
		if code != 201 {
			t.Fatalf("register %s: %d %v", email, code, out)
		}
		return num(t, out, "userId")
	}
	paulID := mkUser("Pending Paul", "paul.pending@example.com")
	quinnID := mkUser("Pending Quinn", "quinn.pending@example.com")

	t.Run("admin sees pending registrations newest first", func(t *testing.T) {
		code, list := doReqList(t, ts, "GET", "/api/registrations", adminToken, nil)
		if code != 200 || len(list) != 2 {
			t.Fatalf("pending list: %d %v", code, list)
		}
		first := list[0].(map[string]any)
		for _, key := range []string{"id", "name", "email", "registeredAt", "status"} {
			if _, ok := first[key]; !ok {
				t.Fatalf("missing field %q in %v", key, first)
			}
		}
		if first["email"] != "quinn.pending@example.com" || first["status"] != "Pending" {
			t.Fatalf("newest-first order or status wrong: %v", list)
		}
	})

	t.Run("registrations require admin authentication", func(t *testing.T) {
		if code, _ := doReq(t, ts, "GET", "/api/registrations", "", nil); code != 401 {
			t.Fatalf("unauthenticated list: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/registrations/"+istr(int64(quinnID))+"/dismiss", "", nil); code != 401 {
			t.Fatalf("unauthenticated dismiss: %d", code)
		}
	})

	t.Run("dismiss hides user for this workspace only and keeps the account", func(t *testing.T) {
		if code, out := doReq(t, ts, "POST", "/api/registrations/"+istr(int64(quinnID))+"/dismiss", adminToken, nil); code != 200 {
			t.Fatalf("dismiss: %d %v", code, out)
		}
		_, list := doReqList(t, ts, "GET", "/api/registrations", adminToken, nil)
		for _, item := range list {
			if item.(map[string]any)["email"] == "quinn.pending@example.com" {
				t.Fatal("dismissed user is still pending")
			}
		}
		var dismissals int
		if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM dismissed_registrations WHERE user_id=$1`, int64(quinnID)).Scan(&dismissals); err != nil {
			t.Fatal(err)
		}
		if dismissals != 1 {
			t.Fatalf("expected exactly 1 dismissal row, got %d", dismissals)
		}
		var accountExists bool
		if err := d.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, int64(quinnID)).Scan(&accountExists); err != nil {
			t.Fatal(err)
		}
		if !accountExists {
			t.Fatal("dismissal must never delete the user account")
		}
		if code, _ := doReq(t, ts, "POST", "/api/registrations/"+istr(int64(quinnID))+"/dismiss", adminToken, nil); code != 200 {
			t.Fatalf("re-dismiss should be idempotent, got %d", code)
		}
	})

	t.Run("dismiss validation", func(t *testing.T) {
		if code, _ := doReq(t, ts, "POST", "/api/registrations/999999/dismiss", adminToken, nil); code != 404 {
			t.Fatalf("unknown id should be 404, got %d", code)
		}
	})

	t.Run("add to team moves user out of pending into members", func(t *testing.T) {
		if code, out := doReq(t, ts, "POST", "/api/members", adminToken, map[string]any{"email": "paul.pending@example.com"}); code != 201 {
			t.Fatalf("add paul: %d %v", code, out)
		}
		_, list := doReqList(t, ts, "GET", "/api/registrations", adminToken, nil)
		if len(list) != 0 {
			t.Fatalf("paul should no longer be pending: %v", list)
		}
		_, members := doReqList(t, ts, "GET", "/api/members", adminToken, nil)
		found := false
		for _, m := range members {
			if m.(map[string]any)["email"] == "paul.pending@example.com" {
				found = true
			}
		}
		if !found {
			t.Fatal("paul missing from team members list")
		}
		if code, _ := doReq(t, ts, "POST", "/api/registrations/"+istr(int64(paulID))+"/dismiss", adminToken, nil); code != 409 {
			t.Fatalf("dismissing a member should be 409, got %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/members", adminToken, map[string]any{"email": "paul.pending@example.com"}); code != 409 {
			t.Fatalf("duplicate add should be 409, got %d", code)
		}
	})

	t.Run("member cannot see or dismiss registrations", func(t *testing.T) {
		code, out := doReq(t, ts, "POST", "/api/auth/login", "", map[string]any{"email": "paul.pending@example.com", "password": "password123"})
		if code != 200 {
			t.Fatalf("paul login: %d", code)
		}
		paulToken, _ := out["token"].(string)
		if code, _ := doReq(t, ts, "GET", "/api/registrations", paulToken, nil); code != 403 {
			t.Fatalf("member registrations access: %d", code)
		}
		if code, _ := doReq(t, ts, "POST", "/api/registrations/"+istr(int64(quinnID))+"/dismiss", paulToken, nil); code != 403 {
			t.Fatalf("member dismiss: %d", code)
		}
	})

	t.Run("dismissal is workspace-scoped", func(t *testing.T) {
		code, out := doReq(t, ts, "POST", "/api/auth/register", "", map[string]any{"name": "Solo Admin", "email": "solo@example.com", "password": "password123", "confirmPassword": "password123"})
		if code != 201 {
			t.Fatalf("register solo admin: %d %v", code, out)
		}
		soloID := num(t, out, "userId")
		var soloWS int64
		if err := d.Pool.QueryRow(ctx, `INSERT INTO workspaces(name,admin_user_id) VALUES('Solo Workspace',$1) RETURNING id`, int64(soloID)).Scan(&soloWS); err != nil {
			t.Fatal(err)
		}
		if _, err := d.Pool.Exec(ctx, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'Administrator')`, soloWS, int64(soloID)); err != nil {
			t.Fatal(err)
		}
		code, out = doReq(t, ts, "POST", "/api/auth/login", "", map[string]any{"email": "solo@example.com", "password": "password123"})
		if code != 200 {
			t.Fatalf("solo login: %d %v", code, out)
		}
		soloToken, _ := out["token"].(string)

		// Quinn was dismissed by the first workspace but must still be pending
		// for this unrelated workspace.
		_, soloList := doReqList(t, ts, "GET", "/api/registrations", soloToken, nil)
		found := false
		for _, item := range soloList {
			if item.(map[string]any)["email"] == "quinn.pending@example.com" {
				found = true
			}
		}
		if !found {
			t.Fatal("dismissal from another workspace must not hide quinn here")
		}
		if code, _ := doReq(t, ts, "POST", "/api/registrations/"+istr(int64(quinnID))+"/dismiss", soloToken, nil); code != 200 {
			t.Fatalf("solo dismiss: %d", code)
		}
		// The second dismissal is a separate row; the first workspace keeps
		// exactly one dismissal for quinn.
		var firstWSDismissals int
		if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM dismissed_registrations WHERE user_id=$1 AND workspace_id=$2`, int64(quinnID), adminWS).Scan(&firstWSDismissals); err != nil {
			t.Fatal(err)
		}
		if firstWSDismissals != 1 {
			t.Fatalf("first workspace should still have exactly 1 dismissal, got %d", firstWSDismissals)
		}
	})

	t.Run("new registration appears in pending list", func(t *testing.T) {
		mkUser("Riley Recent", "riley.recent@example.com")
		_, list := doReqList(t, ts, "GET", "/api/registrations", adminToken, nil)
		if len(list) != 1 || list[0].(map[string]any)["email"] != "riley.recent@example.com" {
			t.Fatalf("expected only riley pending for admin, got %v", list)
		}
	})
}
