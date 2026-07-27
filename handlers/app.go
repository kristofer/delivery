package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gorilla/sessions"
	"github.com/kristofer/delivery/models"
)

const (
	sessionName    = "deliver-session"
	sessionUserID  = "user_id"      // int64 – GitHub user ID in users table
	sessionAdminID = "admin_id"     // int64 – local_admin ID
	sessionIsAdmin = "is_local_admin"
)

// contextKey is a private type for context keys.
type contextKey string

const ctxUserKey contextKey = "user"
const ctxAdminKey contextKey = "local_admin"

// App holds shared dependencies for all handlers.
type App struct {
	Store   *sessions.CookieStore
	Users   *models.UserStore
	Cohorts *models.CohortStore
	Weeks   *models.WeekStore
	Acts    *models.ActivityStore
	Subs    *models.SubmissionStore
	Groups  *models.GroupStore
}

// currentUser extracts the User from context (set by RequireAuth).
func currentUser(r *http.Request) *models.User {
	u, _ := r.Context().Value(ctxUserKey).(*models.User)
	return u
}

// isLocalAdmin returns true if the request is from a local admin session.
func isLocalAdmin(r *http.Request) bool {
	v, _ := r.Context().Value(ctxAdminKey).(bool)
	return v
}

// RequireAuth middleware loads the session user and rejects unauthenticated requests.
func (a *App) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := a.Store.Get(r, sessionName)
		if err != nil || sess.IsNew {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Local admin session
		if isAdm, _ := sess.Values[sessionIsAdmin].(bool); isAdm {
			adminID, _ := sess.Values[sessionAdminID].(int64)
			if adminID == 0 {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			ctx := context.WithValue(r.Context(), ctxAdminKey, true)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// GitHub user session
		rawID := sess.Values[sessionUserID]
		if rawID == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		var userID int64
		switch v := rawID.(type) {
		case int64:
			userID = v
		case string:
			userID, _ = strconv.ParseInt(v, 10, 64)
		}

		user, err := a.Users.GetUserByID(userID)
		if err == sql.ErrNoRows || user == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		if !user.Whitelisted {
			http.Error(w, "Your account is not yet approved. Please contact an administrator.", http.StatusForbidden)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin middleware ensures the user is an admin (local or GitHub admin role).
func (a *App) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isLocalAdmin(r) {
			next.ServeHTTP(w, r)
			return
		}
		u := currentUser(r)
		if u == nil || u.Role != models.RoleAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireInstructor ensures the user is an instructor or admin.
func (a *App) RequireInstructor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isLocalAdmin(r) {
			next.ServeHTTP(w, r)
			return
		}
		u := currentUser(r)
		if u == nil || (u.Role != models.RoleAdmin && u.Role != models.RoleInstructor) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
