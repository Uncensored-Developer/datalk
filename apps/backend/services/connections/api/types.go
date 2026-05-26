package api

import "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"

type NewConnectionParams struct {
	Name     string
	Database connections.Database
	DSN      string
	UserID   int32
	Metadata connections.Metadata
}

type UpdateConnectionParams struct {
	ID        int32
	Name      *string
	Database  *connections.Database
	DSN       *string
	IsEnabled *bool
	Metadata  *connections.Metadata
}

type ListConnectionsParams struct {
	UserID  int32
	IsAdmin bool
}

type NewAccessParams struct {
	UserID       int32
	ConnectionID int32
	CanQuery     bool
	AllowWrites  bool
	CanManage    bool
}

type ListAccessParams struct {
	UserID       []int32
	ConnectionID []int32
}
