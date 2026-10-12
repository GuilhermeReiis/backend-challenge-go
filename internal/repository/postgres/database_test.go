package postgres

import (
	"context"
	"net/url"
	"testing"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/config"
	"go.uber.org/fx"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func TestDatabaseURL(t *testing.T) {
	t.Run("preserves credentials containing reserved characters", func(t *testing.T) {
		cfg := config.Database{
			Host: "::1", Port: 15432, Name: "wagering",
			User: "app@user", Password: "test-only:@/?#% password", SSLMode: "require",
		}

		parsed, err := url.Parse(databaseURL(cfg))
		if err != nil {
			t.Fatal("expected a valid connection URL")
		}

		password, present := parsed.User.Password()
		if !present || password != cfg.Password || parsed.User.Username() != cfg.User {
			t.Error("expected credentials to survive URL encoding")
		}
		if parsed.Host != "[::1]:15432" || parsed.Path != "/wagering" ||
			parsed.Query().Get("sslmode") != "require" {
			t.Error("expected host, database and ssl mode to be preserved")
		}
	})
}

func TestNewDatabase(t *testing.T) {
	t.Run("closes the pool when startup is canceled", func(t *testing.T) {
		lifecycle := &testLifecycle{}
		cfg := config.Config{Database: config.Database{
			Host: "127.0.0.1", Port: 15432, Name: "wagering",
			User: "wagering_app", Password: "test-only", SSLMode: "disable",
		}}

		db, err := NewDatabase(lifecycle, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if len(lifecycle.hooks) != 1 {
			t.Fatal("expected one database lifecycle hook")
		}
		t.Cleanup(func() {
			lifecycle.hooks[0].OnStop(context.Background())
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err = lifecycle.hooks[0].OnStart(ctx)
		if err == nil {
			t.Fatal("expected startup cancellation to fail the connection check")
		}

		pool, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		err = pool.PingContext(context.Background())
		if err == nil || err.Error() != "sql: database is closed" {
			t.Fatal("expected the pool to be closed after startup failure")
		}
	})
}
