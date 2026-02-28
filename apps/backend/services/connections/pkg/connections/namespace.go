package connections

import "time"

type NamespaceType string

const (
	NamespaceTypeSchema   NamespaceType = "schema"
	NamespaceTypeDatabase NamespaceType = "database"
	NamespaceTypeKeyspace NamespaceType = "keyspace"
)

type Namespace struct {
	ID            int32
	ConnectionID  int32
	Name          string
	NamespaceType NamespaceType
	IsEnabled     bool
	CreatedAt     time.Time
}
