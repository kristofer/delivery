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
