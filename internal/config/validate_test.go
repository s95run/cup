package config

import "testing"

func TestValidateSecretLength(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Addr: ":8080"},
		DB:     DBConfig{Schema: "cup"},
		Auth:   AuthConfig{JWTSecret: "short"},
		S95:    S95Config{SSOSecret: "change-me-sso-secret-min-32-chars-long"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected jwt secret length error")
	}
	cfg.Auth.JWTSecret = "jwt-secret-min-32-chars-xxxxxxxxxx"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}