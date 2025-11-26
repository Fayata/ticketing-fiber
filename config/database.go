package config

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2/middleware/session"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB    *gorm.DB
	Store *session.Store
)

func InitDatabase(cfg *Config) error {
	var err error

	logLevel := logger.Error
	// logLevel := logger.Silent

	// Connect to SQLite
	DB, err = gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})

	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	log.Println("✅ Database connected successfully")
	return nil
}

func AutoMigrate(models ...interface{}) error {
	if err := DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	// log.Println("✅ Database migration completed")
	return nil
}
