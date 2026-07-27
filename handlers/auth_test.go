package handlers

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/kristofer/delivery/db"
	"github.com/kristofer/delivery/models"
)

func TestBootstrapLocalAdminFromEnvCreatesAdmin(t *testing.T) {
	f, err := os.CreateTemp("", "delivery-auth-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	database, err := db.Open(f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	app := &App{
		Users: &models.UserStore{DB: database},
	}

	t.Setenv(localAdminUsernameEnv, "bootstrap-admin")
	t.Setenv(localAdminPasswordEnv, "bootstrap-password")

	created, err := app.bootstrapLocalAdminFromEnv()
	if err != nil {
		t.Fatalf("bootstrapLocalAdminFromEnv() error = %v", err)
	}
	if !created {
		t.Fatal("bootstrapLocalAdminFromEnv() created = false, want true")
	}

	admin, err := app.Users.GetLocalAdmin("bootstrap-admin")
	if err != nil {
		t.Fatalf("GetLocalAdmin() error = %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte("bootstrap-password")); err != nil {
		t.Fatalf("password hash does not match bootstrap password: %v", err)
	}
}

func TestBootstrapLocalAdminFromEnvSkipsWhenMissingCredentials(t *testing.T) {
	f, err := os.CreateTemp("", "delivery-auth-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	database, err := db.Open(f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	app := &App{
		Users: &models.UserStore{DB: database},
	}

	t.Setenv(localAdminUsernameEnv, "bootstrap-admin")
	t.Setenv(localAdminPasswordEnv, "")

	created, err := app.bootstrapLocalAdminFromEnv()
	if err != nil {
		t.Fatalf("bootstrapLocalAdminFromEnv() error = %v", err)
	}
	if created {
		t.Fatal("bootstrapLocalAdminFromEnv() created = true, want false")
	}

	count, err := app.Users.CountLocalAdmins()
	if err != nil {
		t.Fatalf("CountLocalAdmins() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("CountLocalAdmins() = %d, want 0", count)
	}
}

func TestBootstrapFromEnvCreatesAdminAtStartup(t *testing.T) {
	f, err := os.CreateTemp("", "delivery-bootstrap-fromenv-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	database, err := db.Open(f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	app := &App{
		Users: &models.UserStore{DB: database},
	}

	t.Setenv(localAdminUsernameEnv, "startup-admin")
	t.Setenv(localAdminPasswordEnv, "startup-password")

	if err := app.BootstrapFromEnv(); err != nil {
		t.Fatalf("BootstrapFromEnv() error = %v", err)
	}

	admin, err := app.Users.GetLocalAdmin("startup-admin")
	if err != nil {
		t.Fatalf("GetLocalAdmin() error = %v, want admin to exist after BootstrapFromEnv", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte("startup-password")); err != nil {
		t.Fatalf("password hash does not match: %v", err)
	}
}

func TestBootstrapFromEnvIsIdempotent(t *testing.T) {
	f, err := os.CreateTemp("", "delivery-bootstrap-idempotent-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	database, err := db.Open(f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	app := &App{
		Users: &models.UserStore{DB: database},
	}

	t.Setenv(localAdminUsernameEnv, "idempotent-admin")
	t.Setenv(localAdminPasswordEnv, "idempotent-password")

	// First call: creates the admin.
	if err := app.BootstrapFromEnv(); err != nil {
		t.Fatalf("first BootstrapFromEnv() error = %v", err)
	}

	// Second call: should be a no-op (admin already exists).
	if err := app.BootstrapFromEnv(); err != nil {
		t.Fatalf("second BootstrapFromEnv() error = %v", err)
	}

	count, err := app.Users.CountLocalAdmins()
	if err != nil {
		t.Fatalf("CountLocalAdmins() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("CountLocalAdmins() = %d, want 1 after two BootstrapFromEnv calls", count)
	}
}
