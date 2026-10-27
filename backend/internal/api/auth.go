package api

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"team-task-management/backend/internal/store"
)

type claims struct {
	UserID      int64  `json:"uid"`
	WorkspaceID int64  `json:"wid"`
	Role        string `json:"role"`
	jwt.RegisteredClaims
}
type ctxKey string

const userKey ctxKey = "user"

type identity struct {
	UserID, WorkspaceID int64
	Role, Name, Email   string
}
type registerRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Burned on lookups of unknown emails so login takes comparable time whether
// or not the account exists (prevents user-enumeration via response timing).
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-not-a-password"), bcrypt.DefaultCost)

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in registerRequest
	if decode(r, &in) != nil {
		fail(w, 400, "Invalid request body")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = store.NormalizeEmail(in.Email)
	if len(in.Name) < 2 || len(in.Name) > 120 || !emailPattern.MatchString(in.Email) || len(in.Password) < 8 || in.Password != in.ConfirmPassword {
		fail(w, 400, "Enter a valid name and email; password must be at least 8 characters and match confirmation")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "Could not secure password")
		return
	}
	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		fail(w, 500, "Database error")
		return
	}
	defer tx.Rollback(r.Context())
	// Serialize the first-user decision so concurrent registrations cannot both
	// observe an empty users table and each claim an administrator workspace.
	if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(846201)`); err != nil {
		fail(w, 500, "Database error")
		return
	}
	var existing int
	if err = tx.QueryRow(r.Context(), `SELECT count(*) FROM users`).Scan(&existing); err != nil {
		fail(w, 500, "Database error")
		return
	}
	var uid int64
	if err = tx.QueryRow(r.Context(), `INSERT INTO users(name,email,password_hash) VALUES($1,$2,$3) RETURNING id`, in.Name, in.Email, string(hash)).Scan(&uid); err != nil {
		fail(w, 409, "Email is already registered or account could not be created")
		return
	}
	if existing > 0 {
		if err = tx.Commit(r.Context()); err != nil {
			fail(w, 500, "Could not finish registration")
			return
		}
		writeJSON(w, 201, map[string]any{"message": "Registration successful. Ask the workspace administrator to add your registered email before accessing workspace data.", "userId": uid, "workspaceId": nil, "role": "Unassigned"})
		return
	}
	var wid int64
	if err = tx.QueryRow(r.Context(), `INSERT INTO workspaces(name,admin_user_id) VALUES($1,$2) RETURNING id`, in.Name+"'s Workspace", uid).Scan(&wid); err != nil {
		fail(w, 500, "Could not create workspace")
		return
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'Administrator')`, wid, uid); err != nil {
		fail(w, 500, "Could not create workspace membership")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 500, "Could not finish registration")
		return
	}
	token, err := s.makeToken(uid, wid, "Administrator")
	if err != nil {
		fail(w, 500, "Could not create session; configure JWT_SECRET with at least 32 characters")
		return
	}
	writeJSON(w, 201, map[string]any{"token": token, "user": map[string]any{"id": uid, "name": in.Name, "email": in.Email, "role": "Administrator", "workspaceId": wid}})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if decode(r, &in) != nil {
		fail(w, 400, "Invalid request body")
		return
	}
	in.Email = store.NormalizeEmail(in.Email)
	var uid int64
	var name, email, hash string
	err := s.db.Pool.QueryRow(r.Context(), `SELECT id,name,email,password_hash FROM users WHERE email=$1`, in.Email).Scan(&uid, &name, &email, &hash)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(in.Password))
		fail(w, 401, "Invalid email or password")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		fail(w, 401, "Invalid email or password")
		return
	}
	var wid int64
	var role string
	err = s.db.Pool.QueryRow(r.Context(), `SELECT workspace_id,role FROM workspace_members WHERE user_id=$1`, uid).Scan(&wid, &role)
	if err != nil {
		fail(w, 403, "Your account is not currently a member of a workspace. Contact the administrator.")
		return
	}
	token, err := s.makeToken(uid, wid, role)
	if err != nil {
		fail(w, 500, "Could not create session")
		return
	}
	writeJSON(w, 200, map[string]any{"token": token, "user": map[string]any{"id": uid, "name": name, "email": email, "role": role, "workspaceId": wid}})
}
func (s *Server) makeToken(uid, wid int64, role string) (string, error) {
	if len(s.cfg.JWTSecret) < 32 {
		return "", http.ErrServerClosed
	}
	c := claims{uid, wid, role, jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.cfg.JWTTTLHours) * time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now()), Issuer: "team-task-management"}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(s.cfg.JWTSecret))
}
func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearer(r)
		if raw == "" {
			fail(w, 401, "Authentication required")
			return
		}
		t, err := jwt.ParseWithClaims(raw, &claims{}, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(s.cfg.JWTSecret), nil
		})
		if err != nil || !t.Valid {
			fail(w, 401, "Invalid or expired token")
			return
		}
		c := t.Claims.(*claims)
		var i identity
		err = s.db.Pool.QueryRow(r.Context(), `SELECT u.id,u.name,u.email,wm.workspace_id,wm.role FROM users u JOIN workspace_members wm ON wm.user_id=u.id WHERE u.id=$1 AND wm.workspace_id=$2`, c.UserID, c.WorkspaceID).Scan(&i.UserID, &i.Name, &i.Email, &i.WorkspaceID, &i.Role)
		if err != nil {
			fail(w, 403, "Workspace membership is required")
			return
		}
		i.UserID = c.UserID
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, i)))
	})
}
func current(r *http.Request) identity { i, _ := r.Context().Value(userKey).(identity); return i }
func (s *Server) admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if current(r).Role != "Administrator" {
			fail(w, 403, "Administrator permission required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	i := current(r)
	writeJSON(w, 200, map[string]any{"id": i.UserID, "name": i.Name, "email": i.Email, "role": i.Role, "workspaceId": i.WorkspaceID})
}
