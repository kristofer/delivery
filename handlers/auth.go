package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"golang.org/x/crypto/bcrypt"

	"github.com/kristofer/delivery/config"
)

const (
	localAdminUsernameEnv = "LOCAL_ADMIN_USERNAME"
	localAdminPasswordEnv = "LOCAL_ADMIN_PASSWORD"
)

// InitOAuth configures GitHub OAuth via goth.
func InitOAuth(clientID, clientSecret, callbackURL string) {
	goth.UseProviders(
		github.New(clientID, clientSecret, callbackURL, "user:email"),
	)
}

func (a *App) bootstrapLocalAdminFromEnv() (bool, error) {
	count, err := a.Users.CountLocalAdmins()
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}

	username := os.Getenv(localAdminUsernameEnv)
	password := os.Getenv(localAdminPasswordEnv)
	if username == "" || password == "" {
		return false, nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	if err := a.Users.CreateLocalAdmin(username, string(hash)); err != nil {
		return false, err
	}
	return true, nil
}

// BootstrapFromEnv creates the first local admin from LOCAL_ADMIN_USERNAME /
// LOCAL_ADMIN_PASSWORD environment variables when no local admin exists yet.
// It is safe to call at startup; it is a no-op when an admin already exists or
// when the env vars are not set.
func (a *App) BootstrapFromEnv() error {
	created, err := a.bootstrapLocalAdminFromEnv()
	if err != nil {
		return err
	}
	if created {
		log.Printf("local admin account bootstrapped from environment variables")
	}
	return nil
}

func (a *App) ensureSetupReady(w http.ResponseWriter, r *http.Request) bool {
	count, err := a.Users.CountLocalAdmins()
	if err != nil {
		http.Error(w, "Failed to verify admin status", http.StatusInternalServerError)
		return false
	}
	if count > 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return false
	}

	created, err := a.bootstrapLocalAdminFromEnv()
	if err != nil {
		log.Printf("bootstrap local admin from env failed: %v", err)
		http.Error(w, "Failed to bootstrap admin account", http.StatusInternalServerError)
		return false
	}
	if created {
		http.Redirect(w, r, "/login", http.StatusFound)
		return false
	}
	return true
}

// LoginPage renders the login form.
func (a *App) LoginPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Error": r.URL.Query().Get("error"),
	}
	renderTemplate(w, "login.html", data)
}

// LocalLoginPost handles the local admin login form POST.
func (a *App) LocalLoginPost(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	admin, err := a.Users.GetLocalAdmin(username)
	if err == sql.ErrNoRows || admin == nil {
		http.Redirect(w, r, "/login?error=invalid+credentials", http.StatusFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		http.Redirect(w, r, "/login?error=invalid+credentials", http.StatusFound)
		return
	}

	sess, _ := a.Store.Get(r, sessionName)
	sess.Values[sessionIsAdmin] = true
	sess.Values[sessionAdminID] = admin.ID
	if err := sess.Save(r, w); err != nil {
		http.Error(w, "Could not save session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// GitHubAuthBegin redirects to GitHub OAuth.
func (a *App) GitHubAuthBegin(w http.ResponseWriter, r *http.Request) {
	gothic.BeginAuthHandler(w, r)
}

// GitHubAuthCallback handles the OAuth callback.
func (a *App) GitHubAuthCallback(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Redirect(w, r, "/login?error=oauth+failed", http.StatusFound)
		return
	}

	var ghID int64
	for _, c := range gothUser.RawData {
		if id, ok := c.(float64); ok {
			ghID = int64(id)
			break
		}
	}

	user, err := a.Users.UpsertGitHubUser(
		gothUser.NickName,
		ghID,
		gothUser.Name,
		gothUser.Email,
		gothUser.AvatarURL,
	)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login?error=db+error", http.StatusFound)
		return
	}

	if !user.Whitelisted {
		http.Error(w, "Your GitHub account is not yet approved. Contact an administrator.", http.StatusForbidden)
		return
	}

	sess, _ := a.Store.Get(r, sessionName)
	sess.Values[sessionUserID] = user.ID
	sess.Values[sessionIsAdmin] = false
	if err := sess.Save(r, w); err != nil {
		http.Error(w, "Could not save session", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// Logout clears the session.
func (a *App) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := a.Store.Get(r, sessionName)
	sess.Options.MaxAge = -1
	_ = sess.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// SetupPage shows the initial admin setup form (only if no local admins exist).
func (a *App) SetupPage(w http.ResponseWriter, r *http.Request) {
	if !a.ensureSetupReady(w, r) {
		return
	}
	renderTemplate(w, "setup.html", nil)
}

// SetupPost creates the first local admin.
func (a *App) SetupPost(w http.ResponseWriter, r *http.Request) {
	if !a.ensureSetupReady(w, r) {
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		renderTemplate(w, "setup.html", map[string]interface{}{"Error": "username and password required"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if err := a.Users.CreateLocalAdmin(username, string(hash)); err != nil {
		http.Error(w, "Internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Also set GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET from env if present
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if clientID != "" && clientSecret != "" {
		InitOAuth(clientID, clientSecret, config.OAuthCallbackURL())
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}
