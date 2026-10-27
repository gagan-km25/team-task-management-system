package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"team-task-management/backend/internal/config"
	"team-task-management/backend/internal/store"
)

type Server struct {
	db  *store.DB
	cfg config.Config
	mux *http.ServeMux
}

func New(db *store.DB, cfg config.Config) http.Handler {
	s := &Server{db: db, cfg: cfg, mux: http.NewServeMux()}
	s.routes()
	return s.cors(s.mux)
}
func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	s.mux.HandleFunc("POST /api/auth/register", s.register)
	s.mux.HandleFunc("POST /api/auth/login", s.login)
	s.mux.Handle("GET /api/me", s.auth(http.HandlerFunc(s.me)))
	s.mux.Handle("GET /api/dashboard", s.auth(http.HandlerFunc(s.dashboard)))
	s.mux.Handle("GET /api/members", s.auth(http.HandlerFunc(s.members)))
	s.mux.Handle("POST /api/members", s.auth(s.admin(http.HandlerFunc(s.addMember))))
	s.mux.Handle("DELETE /api/members/{id}", s.auth(s.admin(http.HandlerFunc(s.removeMember))))
	s.mux.Handle("GET /api/projects", s.auth(http.HandlerFunc(s.projects)))
	s.mux.Handle("POST /api/projects", s.auth(s.admin(http.HandlerFunc(s.createProject))))
	s.mux.Handle("GET /api/projects/{id}", s.auth(http.HandlerFunc(s.getProject)))
	s.mux.Handle("PUT /api/projects/{id}", s.auth(s.admin(http.HandlerFunc(s.updateProject))))
	s.mux.Handle("DELETE /api/projects/{id}", s.auth(s.admin(http.HandlerFunc(s.deleteProject))))
	s.mux.Handle("GET /api/tasks", s.auth(http.HandlerFunc(s.tasks)))
	s.mux.Handle("POST /api/tasks", s.auth(s.admin(http.HandlerFunc(s.createTask))))
	s.mux.Handle("PUT /api/tasks/{id}", s.auth(http.HandlerFunc(s.updateTask)))
	s.mux.Handle("DELETE /api/tasks/{id}", s.auth(s.admin(http.HandlerFunc(s.deleteTask))))
}
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == s.cfg.FrontendOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
func decode(r *http.Request, dst any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(dst)
}
func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return ""
}
