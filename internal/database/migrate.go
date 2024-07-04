package database

import (
	"database/sql"
	"github.com/pressly/goose/v3"
	"github.com/sh3ll3y/promotion-service/internal/logging"
	"go.uber.org/zap"
	"time"
)

func RunMigrations(db *sql.DB, migrationsDir string) error {
	var err error
	for i := 0; i < 5; i++ {
		err = runMigrationsOnce(db, migrationsDir)
		if err == nil {
			return nil
		}
		logging.Logger.Warn("Failed to run migrations, retrying...", zap.Error(err), zap.Int("attempt", i+1))
		time.Sleep(5 * time.Second)
	}
	return err
}

func runMigrationsOnce(db *sql.DB, migrationsDir string) error {
	goose.SetBaseFS(nil) // Use the local filesystem
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return err
	}

	logging.Logger.Info("Migrations completed successfully")
	return nil
}