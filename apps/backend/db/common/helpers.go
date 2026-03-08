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
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdobak/go-xerrors"
)

var ErrMigrationsNotFound = xerrors.New("migrations not found")

func DropTestSchema(conn *sql.DB, schema string) error {
	_, err := conn.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS test%s CASCADE;", schema))
	return err
}

func DBFromConfig(cfg config.Config, schema string, migrateUp bool, log *slog.Logger) (*sql.DB, error) {
	hostname := cfg.DBHost
	if hostname == "" {
		return nil, ErrNoDBConfiguration
	}

	SQLConfig := []any{
		cfg.DBUser,
		cfg.DBPassword,
		hostname,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
		schema,
	}
	connectionString := fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s search_path=%s",
		SQLConfig...,
	)

	conn, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}
	conn.SetConnMaxLifetime(5 * time.Minute)

	if _, err := conn.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
		return nil, err
	}

	if _, err := conn.Exec(fmt.Sprintf("SET search_path TO %s", schema)); err != nil {
		return nil, err
	}

	if migrateUp {
		if err := MigrateUp(conn, schema, cfg.GoMigrateTable, log); err != nil {
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

func MigrateUp(conn *sql.DB, schema string, migrationsTable string, log *slog.Logger) error {
	driver, err := postgres.WithInstance(conn, &postgres.Config{
		SchemaName:      schema,
		MigrationsTable: migrationsTable,
	})
	if err != nil {
		return err
	}

	location, err := FindMigrations()
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(location, "postgres", driver)
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
