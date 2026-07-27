package config

import (
	"os"
	"strings"
)

func Addr() string {
	if addr := os.Getenv("ADDR"); addr != "" {
		return addr
	}
	if port := os.Getenv("PORT"); port != "" {
		if strings.HasPrefix(port, ":") {
			return port
		}
		return ":" + port
	}
	return ":8080"
}

func PublicURL() string {
	if publicURL := os.Getenv("APP_URL"); publicURL != "" {
		return strings.TrimRight(publicURL, "/")
	}
	return publicURLFromAddr(Addr())
}

func OAuthCallbackURL() string {
	if callbackURL := os.Getenv("GITHUB_CALLBACK_URL"); callbackURL != "" {
		return callbackURL
	}
	return PublicURL() + "/auth/github/callback"
}

func SessionSecure() bool {
	if os.Getenv("SESSION_SECURE") == "true" {
		return true
	}
	return strings.HasPrefix(PublicURL(), "https://")
}

func publicURLFromAddr(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
}
