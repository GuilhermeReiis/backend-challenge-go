//go:build integration

package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/config"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func TestDatabaseLifecycle(t *testing.T) {
	t.Run("connects with the application role and closes on shutdown", func(t *testing.T) {
		filename := filepath.Join("..", "..", "..", ".env")
		values, err := godotenv.Read(filename)

		if err != nil && !os.IsNotExist(err) {
			t.Fatal("could not read integration environment")
		}

		for _, name := range []string{
			"APP_DB_HOST", "APP_DB_PORT", "POSTGRES_DB",
			"APP_DB_USER", "APP_DB_PASSWORD", "APP_DB_SSLMODE",
		} {
			value, present := os.LookupEnv(name)

			if !present {
				value = values[name]
			}

			t.Setenv(name, value)
		}

		var db *gorm.DB
		app := fx.New(
			fx.Provide(config.Load, postgres.NewDatabase),
			fx.Populate(&db),
			fx.NopLogger,
		)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.Start(ctx); err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer stopCancel()
			app.Stop(stopCtx)
		})

		var identity struct {
			User     string
			Database string
		}

		if err := db.WithContext(ctx).Raw(
			"SELECT current_user AS \"user\", current_database() AS database",
		).Scan(&identity).Error; err != nil {
			t.Fatal("could not verify database identity")
		}

		if identity.User != "wagering_app" || identity.Database != os.Getenv("POSTGRES_DB") {
			t.Fatal("expected a connection with the application role and configured database")
		}

		pool, err := db.DB()

		if err != nil {
			t.Fatal(err)
		}

		if pool.Stats().MaxOpenConnections != 10 {
			t.Error("expected the configured connection pool limit")
		}

		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()

		if err := app.Stop(stopCtx); err != nil {
			t.Fatal(err)
		}

		if err := pool.PingContext(ctx); err == nil || err.Error() != "sql: database is closed" {
			t.Fatal("expected shutdown to close the connection pool")
		}
	})
}
