package config

import (
	"os"
	"time"
)

type Config struct {
	// Server
	Port string

	// Database
	DatabasePath string

	// Email
	EmailHost     string
	EmailPort     int
	EmailUsername string
	EmailPassword string
	EmailFrom     string

	// Session
	SessionSecret string
	SessionExpiry time.Duration

	// App
	AppName string
	Debug   bool
}

func LoadConfig() *Config {
	return &Config{
		Port:          getEnv("PORT", "3000"),
		DatabasePath:  getEnv("DB_PATH", "./ticketing.db"),
		EmailHost:     getEnv("EMAIL_HOST", "cloudtech.id"),
		EmailPort:     587,
		EmailUsername: getEnv("EMAIL_USER", "daffa@cloudtech.id"),
		EmailPassword: getEnv("EMAIL_PASSWORD", ""),
		EmailFrom:     getEnv("EMAIL_FROM", "daffa@cloudtech.id"),
		SessionSecret: getEnv("SESSION_SECRET", "your-secret-key-change-in-production"),
		SessionExpiry: 24 * time.Hour,
		AppName:       "Ticketing System",
		Debug:         getEnv("DEBUG", "true") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
