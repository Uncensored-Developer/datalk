package connections

import "time"

type Database string

const (
	DatabasePostgres Database = "postgres"
	DatabaseMySQL    Database = "mysql"
	DatabaseCQL      Database = "cql"
)

type Connection struct {
	ID        int32
	UserID    int32
	Name      string
	Database  Database
	DSN       string
	IsEnabled bool
	CreatedAt time.Time
}
