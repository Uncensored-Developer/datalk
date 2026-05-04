package schemas

import (
	"context"
	"database/sql"
	"time"

	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	internalschemas "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector"
	_ "github.com/lib/pq"
	"github.com/mdobak/go-xerrors"
)

const (
	introspectionConnMaxLifetime = 3 * time.Minute
	introspectionMaxOpenConns    = 1
	introspectionMaxIdleConns    = 1
)

type (
	openSQLFunc         func(driverName, dsn string) (*sql.DB, error)
	introspectorBuilder func(*sql.DB) (introspector.Introspector, error)
)

type databaseIntrospectorSpec struct {
	kind       introspector.DBKind
	driverName string
	builder    introspectorBuilder
}

var supportedDatabaseSpecs = map[connectiontypes.Database]databaseIntrospectorSpec{
	connectiontypes.DatabasePostgres: {
		kind:       introspector.DBPostgres,
		driverName: "postgres",
		builder:    buildPostgresIntrospector,
	},
	connectiontypes.DatabaseMySQL: {
		kind:       introspector.DBMySQL,
		driverName: "mysql",
		builder:    buildMySQLIntrospector,
	},
}

func buildPostgresIntrospector(db *sql.DB) (introspector.Introspector, error) {
	return introspector.NewPostgres(db)
}

func buildMySQLIntrospector(db *sql.DB) (introspector.Introspector, error) {
	return introspector.NewMySQL(db)
}

type AtlasIntrospectorFactory struct {
	openSQL openSQLFunc
}

func newIntrospectorFactory() *AtlasIntrospectorFactory {
	return &AtlasIntrospectorFactory{
		openSQL: sql.Open,
	}
}

func (f *AtlasIntrospectorFactory) ForConnection(_ context.Context, connection connectiontypes.Connection) (introspector.Introspector, error) {
	if connection.DSN == "" {
		return nil, xerrors.New("connection DSN is required for schema introspection")
	}

	if connection.Database == connectiontypes.DatabaseCQL {
		return nil, introspector.ErrUnsupportedDBKind
	}

	spec, ok := supportedDatabaseSpecs[connection.Database]
	if !ok {
		return nil, xerrors.Newf("unsupported connection database for schema introspection: %s", connection.Database)
	}

	return newConnectionIntrospector(
		spec.kind,
		spec.driverName,
		connection.DSN,
		f.openSQL,
		spec.builder,
	), nil
}

type connectionIntrospector struct {
	kind              introspector.DBKind
	driverName        string
	dsn               string
	openSQL           openSQLFunc
	buildIntrospector introspectorBuilder
}

func newConnectionIntrospector(kind introspector.DBKind, driverName, dsn string, open openSQLFunc, build introspectorBuilder) *connectionIntrospector {
	return &connectionIntrospector{
		kind:              kind,
		driverName:        driverName,
		dsn:               dsn,
		openSQL:           open,
		buildIntrospector: build,
	}
}

func (i *connectionIntrospector) Kind() introspector.DBKind {
	return i.kind
}

func (i *connectionIntrospector) Introspect(ctx context.Context, opts introspector.IntrospectOptions) (*introspector.Catalog, error) {
	db, err := i.openSQL(i.driverName, i.dsn)
	if err != nil {
		return nil, xerrors.Newf("failed to open %s connection for schema introspection: %w", i.kind, err)
	}
	defer db.Close()

	configureTransientIntrospectionPool(db)

	inspector, err := i.buildIntrospector(db)
	if err != nil {
		return nil, xerrors.Newf("failed to build %s introspector: %w", i.kind, err)
	}

	catalog, err := inspector.Introspect(ctx, opts)
	if err != nil {
		return nil, xerrors.Newf("failed to introspect %s schema: %w", i.kind, err)
	}
	return catalog, nil
}

func configureTransientIntrospectionPool(db *sql.DB) {
	// Introspection is a short-lived operation, so keep the transient connection
	// pool narrow and let Atlas drive the actual inspection work.
	db.SetConnMaxLifetime(introspectionConnMaxLifetime)
	db.SetMaxOpenConns(introspectionMaxOpenConns)
	db.SetMaxIdleConns(introspectionMaxIdleConns)
}

var (
	_ internalschemas.IntrospectorFactory = (*AtlasIntrospectorFactory)(nil)
	_ introspector.Introspector           = (*connectionIntrospector)(nil)
)
