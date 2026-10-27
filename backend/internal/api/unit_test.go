package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"team-task-management/backend/internal/config"
	"team-task-management/backend/internal/store"
)

func TestHealthRoute(t *testing.T) {
	// Route construction needs a database in normal app startup; test JSON helper independently.
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestValidateTask(t *testing.T) {
	valid := taskInput{ProjectID: 1, Title: "Write spec", AssignedUserID: 2, Priority: "High", DueDate: "2026-12-01", Status: "To Do"}
	if err := validateTask(valid); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*taskInput)
	}{
		{"missing project", func(x *taskInput) { x.ProjectID = 0 }},
		{"missing assignee", func(x *taskInput) { x.AssignedUserID = 0 }},
		{"title too short", func(x *taskInput) { x.Title = " a " }},
		{"bad priority", func(x *taskInput) { x.Priority = "Urgent" }},
		{"bad status", func(x *taskInput) { x.Status = "Done" }},
		{"bad due date", func(x *taskInput) { x.DueDate = "01/12/2026" }},
	}
	for _, c := range cases {
		in := valid
		c.mutate(&in)
		if err := validateTask(in); err == nil {
			t.Errorf("%s: expected error, got nil", c.name)
		}
	}
	empty := valid
	empty.Status = ""
	if err := validateTask(empty); err != nil {
		t.Errorf("empty status should default to To Do, got %v", err)
	}
}

func TestValidProject(t *testing.T) {
	ok := projectInput{Name: " Apollo ", Description: " desc ", StartDate: "2026-01-01", EndDate: "2026-12-31"}
	in, good := validProject(ok)
	if !good {
		t.Fatal("valid project rejected")
	}
	if in.Name != "Apollo" || in.Description != "desc" {
		t.Fatalf("trimming not applied: %q %q", in.Name, in.Description)
	}
	bad := []projectInput{
		{Name: "A", StartDate: "2026-01-01", EndDate: "2026-12-31"},
		{Name: "Apollo", StartDate: "2026-12-31", EndDate: "2026-01-01"},
		{Name: "Apollo", StartDate: "2026-01-32", EndDate: "2026-12-31"},
		{Name: "Apollo", StartDate: "not-a-date", EndDate: "2026-12-31"},
	}
	for _, in := range bad {
		if _, good := validProject(in); good {
			t.Errorf("expected rejection of %+v", in)
		}
	}
}

func TestEmailPattern(t *testing.T) {
	for _, good := range []string{"a@b.co", "user.name+tag@example.io", "BOB@Example.COM"} {
		if !emailPattern.MatchString(good) {
			t.Errorf("expected %q to match", good)
		}
	}
	for _, bad := range []string{"", "no-at-sign", "a@b", "a b@c.com", "a@b c.com"} {
		if emailPattern.MatchString(bad) {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := store.NormalizeEmail("  Bob@Example.COM "); got != "bob@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"status":"To Do","admin":true}`))
	var dst struct {
		Status string `json:"status"`
	}
	if decode(r, &dst) == nil {
		t.Fatal("unknown fields should be rejected")
	}
	r2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"status":"To Do"}`))
	var dst2 struct {
		Status string `json:"status"`
	}
	if decode(r2, &dst2) != nil || dst2.Status != "To Do" {
		t.Fatal("known fields should decode")
	}
}

func TestMakeTokenRoundTrip(t *testing.T) {
	s := &Server{cfg: config.Config{JWTSecret: strings.Repeat("s", 40), JWTTTLHours: 1}}
	token, err := s.makeToken(7, 3, "Administrator")
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) { return []byte(s.cfg.JWTSecret), nil })
	if err != nil || !parsed.Valid {
		t.Fatalf("token did not parse: %v", err)
	}
	c := parsed.Claims.(*claims)
	if c.UserID != 7 || c.WorkspaceID != 3 || c.Role != "Administrator" {
		t.Fatalf("claims mismatch: %+v", c)
	}
	if c.ExpiresAt == nil || time.Until(c.ExpiresAt.Time) > time.Hour {
		t.Fatalf("token should expire in about an hour")
	}

	short := &Server{cfg: config.Config{JWTSecret: "too-short"}}
	if _, err := short.makeToken(1, 1, "Administrator"); err == nil {
		t.Fatal("short secrets must be rejected")
	}

	other := &Server{cfg: config.Config{JWTSecret: strings.Repeat("x", 40)}}
	parsed, err = jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) { return []byte(other.cfg.JWTSecret), nil })
	if err == nil && parsed.Valid {
		t.Fatal("token signed with another secret must not validate")
	}
}

func TestAuthMiddlewareRejectsMissingAndGarbageTokens(t *testing.T) {
	// No database touch expected before token validation, so Server.db is nil here.
	s := &Server{cfg: config.Config{JWTSecret: strings.Repeat("s", 40)}, mux: http.NewServeMux()}
	handler := s.auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("handler must not run without a valid token") }))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token: expected 401, got %d", w.Code)
	}
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	r2.Header.Set("Authorization", "Bearer not-a-jwt")
	handler.ServeHTTP(w2, r2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("garbage token: expected 401, got %d", w2.Code)
	}
}
