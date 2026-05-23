package storage

import "github.com/mdobak/go-xerrors"

var (
	ErrOwnerAlreadyExists     = xerrors.New("owner already exists")
	ErrRefreshTokenNotFound   = xerrors.New("refresh token not found")
	ErrRefreshTokenNotRevoked = xerrors.New("refresh token not revoked")
)
