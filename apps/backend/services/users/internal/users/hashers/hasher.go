package hashers

import "context"

//go:generate go tool with-modfile mockery --name Hasher --outpkg testing --output ./testing --filename generated__hasher_mocks.go
type Hasher interface {
	Hash(ctx context.Context, password string) (string, error)

	Verify(ctx context.Context, password, hash string) (bool, error)
}
