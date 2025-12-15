package common

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdobak/go-xerrors"
)

var (
	ErrMigrationsNotFound = xerrors.New("migrations not found")
	ErrNoDBConfiguration  = xerrors.New("no DB configuration found")
)

func DropTestSchema(conn *sql.DB, schema string) error {
	_, err := conn.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS test%s CASCADE;", schema))
	return err
}

func DBFromConfig(cfg config.Config, migrateUp bool, log *slog.Logger) (*sql.DB, error) {
	if cfg.DbDSN == "" {
		return nil, ErrNoDBConfiguration
	}

	var conn *sql.DB
	var err error

	conn, err = sql.Open("sqlite3", cfg.DbDSN)
	if err != nil {
		return nil, err
	}
	conn.SetConnMaxLifetime(5 * time.Minute)

	if migrateUp {
		if err := MigrateUp(conn, log); err != nil {
			return conn, err
		}
	}

	return conn, nil
}

func FindMigrations() (string, error) {
	locations := []string{
		// services api tests
		"../../../../../db/migrations",

		"db/migrations",

		// echo tests
		"../../db/migrations",

		"../../../db/migrations",
	}

	for _, location := range locations {
		if _, err := os.Stat(location); !os.IsNotExist(err) {
			return fmt.Sprintf("file://%s", location), nil
		}
	}
	return "", ErrMigrationsNotFound
}

func MigrateUp(conn *sql.DB, log *slog.Logger) error {
	driver, err := sqlite3.WithInstance(conn, &sqlite3.Config{})
	if err != nil {
		return err
	}

	location, err := FindMigrations()
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(location, "sqlite3", driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			log.Error("failed to migrate", logger.Err(err))
			return err
		}

		log.Debug("migrations were up to date")
	} else {
		log.Info("migrations ran successfully")
	}

	return nil
}
