package postgres

import (
	"fmt"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
)

var (
	cfg      = config.MustLoad()
	runner   = common.MustNewTestRunner("pubsub_postgres")
	connInfo = testConnInfo(cfg, runner.Schema)
	bus      = NewBus(runner.Conn, connInfo)
)

func TestMain(m *testing.M) {
	runner.Run(m)
}

func testConnInfo(cfg config.Config, schema string) string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s search_path=test%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
		schema,
	)
}
