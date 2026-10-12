package config_test

import (
	"strings"
	"testing"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/config"
)

func setValidEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("APP_DB_HOST", "127.0.0.1")
	t.Setenv("APP_DB_PORT", "15432")
	t.Setenv("POSTGRES_DB", "wagering")
	t.Setenv("APP_DB_USER", "wagering_app")
	t.Setenv("APP_DB_PASSWORD", "test-only:@/?#% password")
	t.Setenv("APP_DB_SSLMODE", "disable")
}

func TestLoad(t *testing.T) {
	t.Run("loads application credentials independently of admin credentials", func(t *testing.T) {
		setValidEnvironment(t)
		t.Setenv("POSTGRES_USER", "admin-only")
		t.Setenv("POSTGRES_PASSWORD", "admin-only")

		cfg, err := config.Load()

		if err != nil {
			t.Fatal(err)
		}

		if cfg.Database.User != "wagering_app" ||
			cfg.Database.Password != "test-only:@/?#% password" ||
			cfg.Database.Port != 15432 {
			t.Error("expected application credentials and port to be preserved")
		}
	})

	t.Run("rejects missing configuration", func(t *testing.T) {
		for _, name := range []string{
			"APP_DB_HOST", "APP_DB_PORT", "POSTGRES_DB",
			"APP_DB_USER", "APP_DB_PASSWORD", "APP_DB_SSLMODE",
		} {
			t.Run(name, func(t *testing.T) {
				setValidEnvironment(t)
				t.Setenv(name, "")

				if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), name) {
					t.Fatal("expected error identifying the missing variable")
				}
			})
		}
	})

	t.Run("rejects invalid ports without exposing their values", func(t *testing.T) {
		for _, port := range []string{"0", "65536", "-1", "not-a-port"} {
			t.Run(port, func(t *testing.T) {
				setValidEnvironment(t)
				t.Setenv("APP_DB_PORT", port)

				if _, err := config.Load(); err == nil || err.Error() != "config: APP_DB_PORT must be between 1 and 65535" {
					t.Fatal("expected a configuration error without the supplied value")
				}
			})
		}
	})

	t.Run("rejects invalid ssl mode", func(t *testing.T) {
		setValidEnvironment(t)
		t.Setenv("APP_DB_SSLMODE", "invalid")

		if _, err := config.Load(); err == nil || err.Error() != "config: invalid APP_DB_SSLMODE" {
			t.Fatal("expected invalid ssl mode error")
		}
	})
}
