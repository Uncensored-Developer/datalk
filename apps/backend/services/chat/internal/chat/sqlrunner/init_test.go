package sqlrunner

import (
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/stretchr/testify/require"
)

var (
	integrationRunner     *common.TestRunner
	integrationRunnerCfg  config.Config
	integrationRunnerOnce sync.Once
)

func TestMain(m *testing.M) {
	code := m.Run()
	if integrationRunner != nil && integrationRunner.Conn != nil {
		if err := common.DropTestSchema(integrationRunner.Conn, integrationRunner.Schema); err != nil {
			integrationRunner.Logger.Error("failed to drop integration test schema", slog.Any("err", err))
		}
		if err := integrationRunner.Conn.Close(); err != nil {
			integrationRunner.Logger.Error("failed to close integration test connection", slog.Any("err", err))
		}
	}

	os.Exit(code)
}

func requireIntegrationRunner(t *testing.T, schema string) (*common.TestRunner, config.Config) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB_HOST is not configured")
	}

	integrationRunnerOnce.Do(func() {
		integrationRunnerCfg = config.MustLoad()
		integrationRunner = common.MustNewTestRunner(schema + "_integration")
	})
	require.NotNil(t, integrationRunner)
	require.NotNil(t, integrationRunner.Conn)

	return integrationRunner, integrationRunnerCfg
}
