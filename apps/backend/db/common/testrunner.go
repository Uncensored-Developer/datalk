package common

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/logger"
	"github.com/mdobak/go-xerrors"
	"github.com/stephenafamo/bob"
)

type TestRunner struct {
	Conn    *sql.DB
	BobConn bob.DB
	Logger  *slog.Logger
	Schema  string
	dbPath  string
	tempDir string
}

func MustNewTestRunner(schema string) *TestRunner {
	cfg := config.MustLoad()
	sLogger := logger.SetupLogger(cfg)
	runner, err := NewTestRunner(cfg, sLogger, schema, 3)
	if err != nil {
		panic(err)
	}
	return runner
}

func NewTestRunner(cfg config.Config, sLogger *slog.Logger, schema string, maxAttempts int) (*TestRunner, error) {
	if maxAttempts <= 0 {
		return nil, errors.New("maxAttempts must be greater than 0")
	}

	tmpDir, err := os.MkdirTemp("", "pkgdb-*")
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(tmpDir, "test_"+schema+".db")
	dsn := "file:" + filepath.ToSlash(dbPath) + "?journal_mode=WAL"

	for attempt := 0; attempt < maxAttempts; attempt++ {
		conn, err := TestDB(dsn, sLogger)
		if err != nil && !errors.Is(err, ErrNoDBConfiguration) {
			errMsg := fmt.Sprintf("failed to initialise DB (attempt %d/%d)", attempt+1, maxAttempts)
			sLogger.Error(errMsg, logger.Err(err))
			if conn != nil {
				conn.Close()
				err := DropTestDB(conn, tmpDir)
				if err != nil {
					errMsg = fmt.Sprintf("failed to initialise DB (attempt %d/%d)", attempt+1, maxAttempts)
					sLogger.Error(errMsg, logger.Err(err))
				}
			}
			continue
		}

		return &TestRunner{Conn: conn, BobConn: bob.NewDB(conn), Schema: schema, dbPath: dbPath, tempDir: tmpDir, Logger: sLogger}, nil
	}

	return nil, xerrors.Newf("failed to initialise DB after %d attempts", maxAttempts)
}

func (r *TestRunner) Run(m *testing.M) {
	result := 0
	if r.Conn == nil {
		r.Logger.Error("database not available, tests disabled")
	} else {
		result = m.Run()
		err := DropTestDB(r.Conn, r.tempDir)
		if err != nil {
			r.Logger.Error("failed to drop schema", logger.Err(err))
		}
		if err := r.Conn.Close(); err != nil {
			r.Logger.Error("failed to close connection", logger.Err(err))
		}
	}

	os.Exit(result)
}
