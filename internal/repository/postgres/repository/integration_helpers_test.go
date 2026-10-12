//go:build integration

package repository_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/config"
	"github.com/GuilhermeReiis/backend-challenge-go/internal/repository/postgres"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func testID(t *testing.T) string {
	t.Helper()
	var bytes [16]byte
	_, err := rand.Read(bytes[:])
	if err != nil {
		t.Fatal(err)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

func startTestDatabase(t *testing.T, cfg config.Config) *gorm.DB {
	t.Helper()
	var db *gorm.DB
	app := fx.New(fx.Supply(cfg), fx.Provide(postgres.NewDatabase), fx.Populate(&db), fx.NopLogger)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := app.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := app.Stop(ctx)
		if err != nil {
			t.Error(err)
		}
	})
	return db
}

func repositoryTestDatabase(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()
	values, err := godotenv.Read(filepath.Join("..", "..", "..", "..", ".env"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal("could not read integration environment")
	}
	for _, name := range []string{
		"APP_DB_HOST", "APP_DB_PORT", "POSTGRES_DB", "APP_DB_USER", "APP_DB_PASSWORD", "APP_DB_SSLMODE", "POSTGRES_USER", "POSTGRES_PASSWORD",
	} {
		value, present := os.LookupEnv(name)
		if !present {
			value = values[name]
		}
		t.Setenv(name, value)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	adminCfg := cfg
	adminCfg.Database.User = os.Getenv("POSTGRES_USER")
	adminCfg.Database.Password = os.Getenv("POSTGRES_PASSWORD")
	if adminCfg.Database.User == "" || adminCfg.Database.Password == "" {
		t.Fatal("administrative credentials are required to create the isolated test database")
	}
	admin := startTestDatabase(t, adminCfg)
	databaseName := "wagering_test_" + strings.ReplaceAll(testID(t), "-", "")
	err = admin.Exec("CREATE DATABASE \"" + databaseName + "\"").Error
	if err != nil {
		t.Fatal("could not create isolated integration database")
	}
	t.Cleanup(func() {
		err := admin.Exec("DROP DATABASE \"" + databaseName + "\" WITH (FORCE)").Error
		if err != nil {
			t.Error("could not remove the test database created by this test")
		}
	})
	adminCfg.Database.Name = databaseName
	migrationDB := startTestDatabase(t, adminCfg)
	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "..", "migrations", "*.up.sql"))
	if err != nil || len(paths) == 0 {
		t.Fatal("could not find SQL migrations")
	}
	for _, path := range paths {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		err = migrationDB.Exec(string(migration)).Error
		if err != nil {
			t.Fatalf("migration failed: %s", filepath.Base(path))
		}
	}
	err = migrationDB.Exec("GRANT USAGE ON SCHEMA public TO wagering_app").Error
	if err != nil {
		t.Fatal("could not grant test schema access")
	}
	cfg.Database.Name = databaseName
	return startTestDatabase(t, cfg), migrationDB
}
