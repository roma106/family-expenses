package db

import (
	"errors"
	"family-expenses/internal/config"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func ConnectToDB(cfg *config.Config) (*sqlx.DB, error) {
	conn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresHost, cfg.PostgresDB)
	db, err := sqlx.Connect("postgres", conn)
	if err != nil {
		slog.Error("Failed to connect to db: ", "error", err)
		return nil, err
	}
	return db, nil
}

func RunMigrations(db *sqlx.DB) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		slog.Error("Failed to create migrations driver", "error", err)
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		slog.Error("Failed to create migrations", "error", err)
		return err
	}

	err = m.Up()

	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("Database is up to date")
		return nil
	}
	if err != nil {
		slog.Error("Failed to run migrations", "error", err)
		return err
	}

	slog.Info("Migrations went successfully")
	return nil
}
