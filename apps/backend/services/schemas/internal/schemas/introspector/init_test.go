package introspector

import (
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
)

var runner = common.MustNewTestRunner("atlas_storage")

func TestMain(m *testing.M) {
	runner.Run(m)
}
