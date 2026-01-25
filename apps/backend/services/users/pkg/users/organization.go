package users

import "time"

type Organization struct {
	ID        int32
	Name      string
	CreatedAt time.Time
	singleton bool
}
