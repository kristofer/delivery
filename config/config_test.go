package config

import "testing"

func TestAddrUsesPortWhenAddrUnset(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("PORT", "9090")

	if got := Addr(); got != ":9090" {
		t.Fatalf("Addr() = %q, want %q", got, ":9090")
	}
}

func TestAddrPrefersAddrOverPort(t *testing.T) {
	t.Setenv("ADDR", "0.0.0.0:8081")
	t.Setenv("PORT", "9090")

	if got := Addr(); got != "0.0.0.0:8081" {
		t.Fatalf("Addr() = %q, want %q", got, "0.0.0.0:8081")
	}
}

func TestOAuthCallbackURLUsesAppURLByDefault(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("PORT", "")
	t.Setenv("APP_URL", "https://deliver.zipcode.rocks/")
	t.Setenv("GITHUB_CALLBACK_URL", "")

	if got := OAuthCallbackURL(); got != "https://deliver.zipcode.rocks/auth/github/callback" {
		t.Fatalf("OAuthCallbackURL() = %q", got)
	}
}

func TestOAuthCallbackURLFallsBackToListenAddress(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("PORT", "9090")
	t.Setenv("APP_URL", "")
	t.Setenv("GITHUB_CALLBACK_URL", "")

	if got := OAuthCallbackURL(); got != "http://localhost:9090/auth/github/callback" {
		t.Fatalf("OAuthCallbackURL() = %q", got)
	}
}

func TestSessionSecureUsesHTTPSAppURL(t *testing.T) {
	t.Setenv("APP_URL", "https://deliver.zipcode.rocks")
	t.Setenv("SESSION_SECURE", "")

	if !SessionSecure() {
		t.Fatal("SessionSecure() = false, want true")
	}
}
