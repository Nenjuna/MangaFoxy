package config

import (
	"os"
)

type Config struct {
	DBName string
	DBUser string
	DBPass string
	DBHost string
	DBPort string
	Port   string
	Env    string

	OktaIssuer       string
	OktaClientID     string
	OktaClientSecret string
	OktaRedirectURI  string
	SessionSecret    string
}

func Load() *Config {
	return &Config{
		DBName: getEnv("DB_NAME", "mangabase"),
		DBUser: getEnv("DB_USER", "mangabase_user"),
		DBPass: getEnv("DB_PASS", "localpassword"),
		DBHost: getEnv("DB_HOST", "127.0.0.1"),
		DBPort: getEnv("DB_PORT", "5432"),
		Port:   getEnv("PORT", "8080"),
		Env:    getEnv("ENV", "dev"),

		OktaIssuer:       getEnv("OKTA_ISSUER", ""),
		OktaClientID:     getEnv("OKTA_CLIENT_ID", ""),
		OktaClientSecret: getEnv("OKTA_CLIENT_SECRET", ""),
		OktaRedirectURI:  getEnv("OKTA_REDIRECT_URI", "http://localhost:8080/authorization-code/callback"),
		SessionSecret:    getEnv("SESSION_SECRET", "change-me-in-production-32chars!"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
