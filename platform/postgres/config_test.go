package postgres

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestURLEscapesSpecialCharactersInPassword(t *testing.T) {
	cfg := Config{Host: "db", Port: "5432", Name: "users_db", User: "user_service", Password: "p@ss/w:rd?#", SSLMode: "disable"}

	parsed, err := pgxpool.ParseConfig(cfg.url())
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if got := parsed.ConnConfig.Password; got != cfg.Password {
		t.Errorf("password = %q, want %q", got, cfg.Password)
	}
	if got := parsed.ConnConfig.Database; got != "users_db" {
		t.Errorf("database = %q, want users_db", got)
	}
}

func TestStringRedactsPassword(t *testing.T) {
	cfg := Config{Host: "db", Port: "5432", Name: "users_db", User: "user_service", Password: "s3cret"}

	if s := cfg.String(); strings.Contains(s, "s3cret") {
		t.Errorf("String() leaks the password: %s", s)
	}
}
