package errors

import "github.com/mdobak/go-xerrors"

var (
	ErrUnsupportedDatabase      = xerrors.New("unsupported database")
	ErrSnapshotNotFound         = xerrors.New("snapshot not found")
	ErrEmbeddingDisabled        = xerrors.New("snapshot embedding is disabled")
	ErrEmbeddedSnapshotNotReady = xerrors.New("embedded snapshot not ready")
)
