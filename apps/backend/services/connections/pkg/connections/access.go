package connections

import "time"

type Access struct {
	UserID       int32
	ConnectionID int32
	CanQuery     bool
	AllowWrites  bool
	CanManage    bool
	GrantedAt    time.Time
}
