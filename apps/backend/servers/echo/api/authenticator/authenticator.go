package authenticator

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

var ErrUnauthorized = xerrors.New("unauthorized")

//go:generate go tool with-modfile mockery --name Authenticator --outpkg testing --output ./testing --filename generated__athenticator_mocks.go
type Authenticator interface {
	Authenticate(ctx context.Context, accessToken string) (*users.User, error)
}
