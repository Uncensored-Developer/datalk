package schemas

import (
	"os"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pubsub/memory"
)

var memoryPublisher = memory.NewMemoryBus()

func TestMain(m *testing.M) {
	pubsub.RegisterPublisher(memoryPublisher)
	os.Exit(m.Run())
}
