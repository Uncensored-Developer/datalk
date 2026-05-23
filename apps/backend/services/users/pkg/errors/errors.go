package errors

import "github.com/mdobak/go-xerrors"

var (
	ErrUserNotFound           = xerrors.New("user not found")
	ErrUnauthorized           = xerrors.New("unauthorized")
	ErrForbidden              = xerrors.New("forbidden")
	ErrSetupUnavailable       = xerrors.New("setup is unavailable")
	ErrInactiveUser           = xerrors.New("user is inactive")
	ErrPasswordChangeRequired = xerrors.New("password change required")
	ErrRefreshTokenInvalid    = xerrors.New("refresh token invalid")
)
