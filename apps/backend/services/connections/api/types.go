package api

import "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"

type NewConnectionParams struct {
	Name     string
	Database connections.Database
	DSN      string
	UserID   int32
}

type NewAccessParams struct {
	UserID       int32
	ConnectionID int32
	CanQuery     bool
	AllowWrites  bool
	CanManage    bool
}

type NewNamespaceParams struct {
	ConnectionID  int32
	Name          string
	NamespaceType connections.NamespaceType
}
