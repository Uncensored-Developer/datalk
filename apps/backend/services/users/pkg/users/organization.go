package users

import "time"

type Organization struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	singleton bool
}
