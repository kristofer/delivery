package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth/gothic"

	"github.com/kristofer/delivery/db"
	"github.com/kristofer/delivery/handlers"
	"github.com/kristofer/delivery/models"
)

func main() {
	// ── Configuration ────────────────────────────────────────────────────────
	dbPath := getenv("DB_PATH", "deliver.db")
	addr := getenv("ADDR", ":8080")
	sessionSecret := getenv("SESSION_SECRET", "change-me-in-production")

	clientID := getenv("GITHUB_CLIENT_ID", "")
	clientSecret := getenv("GITHUB_CLIENT_SECRET", "")
	callbackURL := getenv("GITHUB_CALLBACK_URL", "http://localhost:8080/auth/github/callback")

	// ── Database ─────────────────────────────────────────────────────────────
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	// ── OAuth ─────────────────────────────────────────────────────────────────
	if clientID != "" && clientSecret != "" {
		handlers.InitOAuth(clientID, clientSecret, callbackURL)
	}

	// gorilla/sessions store used by gothic
	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.MaxAge(86400 * 7) // 7 days
	store.Options.HttpOnly = true
	store.Options.Secure = false // set true in production behind HTTPS
	store.Options.SameSite = http.SameSiteLaxMode
	if os.Getenv("SESSION_SECURE") == "true" {
		store.Options.Secure = true
	}
	gothic.Store = store

	// ── App ───────────────────────────────────────────────────────────────────
	app := &handlers.App{
		Store:   store,
		Users:   &models.UserStore{DB: database},
		Cohorts: &models.CohortStore{DB: database},
		Weeks:   &models.WeekStore{DB: database},
		Acts:    &models.ActivityStore{DB: database},
		Subs:    &models.SubmissionStore{DB: database},
		Groups:  &models.GroupStore{DB: database},
	}

	// ── Router ────────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Public routes
	r.Get("/setup", app.SetupPage)
	r.Post("/setup", app.SetupPost)
	r.Get("/login", app.LoginPage)
	r.Post("/login/local", app.LocalLoginPost)
	r.Get("/auth/{provider}", app.GitHubAuthBegin)
	r.Get("/auth/{provider}/callback", app.GitHubAuthCallback)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(app.RequireAuth)

		r.Get("/", app.Home)
		r.Get("/logout", app.Logout)

		// Cohorts (read: all; write: instructor+)
		r.Get("/cohorts", app.CohortList)
		r.Get("/cohorts/{id}", app.CohortShow)

		r.Group(func(r chi.Router) {
			r.Use(app.RequireInstructor)
			r.Get("/cohorts/new", app.CohortNew)
			r.Post("/cohorts", app.CohortCreate)
			r.Get("/cohorts/{id}/edit", app.CohortEdit)
			r.Post("/cohorts/{id}/edit", app.CohortUpdate)
			r.Post("/cohorts/{id}/delete", app.CohortDelete)
			r.Post("/cohorts/{id}/enroll", app.CohortEnroll)
			r.Post("/cohorts/{id}/unenroll/{uid}", app.CohortUnenroll)
			r.Post("/cohorts/{id}/weeks", app.WeekCreate)
			r.Post("/cohorts/{id}/weeks/{wid}/delete", app.WeekDelete)
			r.Get("/cohorts/{id}/weeks/{wid}/activities/new", app.ActivityNew)
			r.Post("/cohorts/{id}/weeks/{wid}/activities", app.ActivityCreate)
			r.Get("/cohorts/{id}/activities/{aid}/edit", app.ActivityEdit)
			r.Post("/cohorts/{id}/activities/{aid}/edit", app.ActivityUpdate)
			r.Post("/cohorts/{id}/activities/{aid}/delete", app.ActivityDelete)
		})

		// Activities (read: all; submit: student)
		r.Get("/cohorts/{id}/activities/{aid}", app.ActivityShow)
		r.Post("/cohorts/{id}/activities/{aid}/submissions", app.SubmissionCreate)
		r.Post("/cohorts/{id}/activities/{aid}/submissions/{sid}/edit", app.SubmissionUpdate)
		r.Post("/cohorts/{id}/activities/{aid}/submissions/{sid}/delete", app.SubmissionDelete)

		// Groups (manage: instructor+)
		r.Group(func(r chi.Router) {
			r.Use(app.RequireInstructor)
			r.Post("/cohorts/{id}/activities/{aid}/groups", app.GroupCreate)
			r.Post("/cohorts/{id}/activities/{aid}/groups/{gid}/members", app.GroupAddMember)
			r.Post("/cohorts/{id}/activities/{aid}/groups/{gid}/members/{uid}/delete", app.GroupRemoveMember)
		})

		// Admin routes
		r.Group(func(r chi.Router) {
			r.Use(app.RequireAdmin)
			r.Get("/admin", app.AdminDashboard)
			r.Post("/admin/users/{id}/whitelist", app.AdminWhitelistUser)
			r.Post("/admin/users/{id}/revoke", app.AdminRevokeUser)
			r.Post("/admin/users/{id}/role", app.AdminUpdateUserRole)
			r.Post("/admin/local-admins", app.AdminCreateLocalAdmin)
		})
	})

	log.Printf("Deliver listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
