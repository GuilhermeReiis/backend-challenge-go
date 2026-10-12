package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Database Database
}

type Database struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

func Load() (Config, error) {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, errors.New("config: could not read .env")
	}

	required := []string{
		"APP_DB_HOST",
		"APP_DB_PORT",
		"POSTGRES_DB",
		"APP_DB_USER",
		"APP_DB_PASSWORD",
		"APP_DB_SSLMODE",
	}

	for _, name := range required {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			return Config{}, fmt.Errorf("config: %s is required", name)
		}
	}

	port, err := strconv.Atoi(os.Getenv("APP_DB_PORT"))
	if err != nil || port < 1 || port > 65535 {
		return Config{}, errors.New("config: APP_DB_PORT must be between 1 and 65535")
	}

	sslMode := os.Getenv("APP_DB_SSLMODE")
	switch sslMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		return Config{}, errors.New("config: invalid APP_DB_SSLMODE")
	}

	return Config{
		Database: Database{
			Host:     os.Getenv("APP_DB_HOST"),
			Port:     port,
			Name:     os.Getenv("POSTGRES_DB"),
			User:     os.Getenv("APP_DB_USER"),
			Password: os.Getenv("APP_DB_PASSWORD"),
			SSLMode:  sslMode,
		},
	}, nil
}
