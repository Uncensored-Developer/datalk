package db

import (
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
)

var (
	runner = common.MustNewTestRunner("schemas_storage")
	s      = NewStorage(runner.Conn)
)

func TestMain(m *testing.M) {
	runner.Run(m)
}
