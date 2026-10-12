package postgres

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/GuilhermeReiis/backend-challenge-go/internal/config"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase(lifecycle fx.Lifecycle, cfg config.Config) (*gorm.DB, error) {
	dsn := databaseURL(cfg.Database)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, errors.New("postgres: could not initialize database connection")
	}

	pool, err := db.DB()
	if err != nil {
		return nil, errors.New("postgres: could not obtain database pool")
	}

	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)
	pool.SetConnMaxIdleTime(5 * time.Minute)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			err := pool.PingContext(pingCtx)
			if err != nil {
				pool.Close()
				return errors.New("postgres: connection check failed")
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return pool.Close()
		},
	})

	return db, nil
}

func databaseURL(cfg config.Database) string {
	connection := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:   "/" + cfg.Name,
	}

	query := connection.Query()
	query.Set("sslmode", cfg.SSLMode)
	query.Set("connect_timeout", "5")
	query.Set("timezone", "UTC")
	connection.RawQuery = query.Encode()

	return connection.String()
}
