package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mangafoxy/internal/config"
	"mangafoxy/internal/models"
)

var DB *gorm.DB

func Connect(cfg *config.Config) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBPort,
	)

	logLevel := logger.Silent
	if cfg.Env == "dev" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.Manga{},
		&models.Chapter{},
		&models.ViewLog{},
		&models.Update{},
	); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	ensureViewLogIndexes(db)

	DB = db
	log.Println("Database connected and migrated")
}

// ensureViewLogIndexes creates composite indexes on go_viewlog that AutoMigrate
// cannot express via struct tags alone.
func ensureViewLogIndexes(db *gorm.DB) {
	stmts := []string{
		// Used by Flush: WHERE processed = false, GROUP BY content_type, object_id
		`CREATE INDEX IF NOT EXISTS idx_viewlog_flush
		 ON go_viewlog (processed, content_type, object_id)`,

		// Used by Cleanup: WHERE processed = true AND timestamp < cutoff
		`CREATE INDEX IF NOT EXISTS idx_viewlog_cleanup
		 ON go_viewlog (processed, timestamp)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			log.Printf("Warning: could not create viewlog index: %v", err)
		}
	}
}
