package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

// AdminDashboard is the admin home page.
func (a *App) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	users, _ := a.Users.ListUsers()
	cohorts, _ := a.Cohorts.ListCohorts()
	data := map[string]interface{}{
		"Users":       users,
		"Cohorts":     cohorts,
		"CurrentUser": currentUser(r),
		"IsLocalAdmin": isLocalAdmin(r),
	}
	renderTemplate(w, "admin.html", data)
}

// AdminWhitelistUser approves a GitHub user.
func (a *App) AdminWhitelistUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	role := r.FormValue("role")
	if role == "" {
		role = "student"
	}
	_ = a.Users.WhitelistUser(id, role)
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// AdminRevokeUser removes a user's whitelist status.
func (a *App) AdminRevokeUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_ = a.Users.RevokeUser(id)
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// AdminUpdateUserRole changes a user's role.
func (a *App) AdminUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	role := r.FormValue("role")
	_ = a.Users.UpdateUserRole(id, role)
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// AdminCreateLocalAdmin creates an additional local admin account.
func (a *App) AdminCreateLocalAdmin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		http.Redirect(w, r, "/admin?error=username+and+password+required", http.StatusFound)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if err := a.Users.CreateLocalAdmin(username, string(hash)); err != nil {
		http.Redirect(w, r, "/admin?error="+err.Error(), http.StatusFound)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}
