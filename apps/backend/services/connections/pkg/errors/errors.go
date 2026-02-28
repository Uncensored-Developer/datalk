package errors

import "github.com/mdobak/go-xerrors"

var (
	ErrConnectionNotFound = xerrors.New("connection not found")
	ErrAccessNotFound     = xerrors.New("access not found")
	ErrNamespaceNotFound  = xerrors.New("namespace not found")
)
