package postgres

import (
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
)

var (
	runner = common.MustNewTestRunner("distlock_postgres")
	locker = NewDistributedLocker(runner.Conn)
)

func TestMain(m *testing.M) {
	runner.Run(m)
}
