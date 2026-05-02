package embedding

import "context"

//go:generate go tool with-modfile mockery --name Client --outpkg testing --output ./testing --filename generated__client_mocks.go
type Client interface {
	EmbedTexts(ctx context.Context, inputs []string) ([][]float32, error)
}
