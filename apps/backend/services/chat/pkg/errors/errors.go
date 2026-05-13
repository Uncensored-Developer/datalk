package errors

import "github.com/mdobak/go-xerrors"

var (
	ErrConversationNotFound     = xerrors.New("conversation not found")
	ErrMessageNotFound          = xerrors.New("message not found")
	ErrProviderNotAvailable     = xerrors.New("provider not available")
	ErrModelNotAvailable        = xerrors.New("model not available")
	ErrEmbeddedSnapshotNotReady = xerrors.New("embedded snapshot not ready")
	ErrInvalidSQL               = xerrors.New("invalid sql")
	ErrUnsupportedDatabaseKind  = xerrors.New("unsupported database kind")
	ErrMessageExecutionFailed   = xerrors.New("message execution failed")
)
