package config

import "os"

type Config struct {
	Environment string
	HTTPPort    string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Environment: getenv("APP_ENV", "development"),
		HTTPPort:    getenv("HTTP_PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://wegil:postgres@localhost:5432/smart_inventory?sslmode=disable"),
	}
}

func (c Config) Address() string {
	return ":" + c.HTTPPort
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}
