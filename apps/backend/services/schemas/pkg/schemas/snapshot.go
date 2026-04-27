package schemas

import (
	"encoding/json"
	"time"
)

type Snapshot struct {
	ID             int32
	ConnectionID   int32
	SchemaHash     string
	SchemaJSON     json.RawMessage
	IntrospectedAt time.Time
}
