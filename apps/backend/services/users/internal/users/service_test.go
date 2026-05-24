package users

import (
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	hashertesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users/hashers/testing"
)

func newAuthTestService(mockStorage storage.Storage, mockHasher *hashertesting.Hasher) *Service {
	return &Service{
		Base:    &base.Base{},
		storage: mockStorage,
		hasher:  mockHasher,
		tokens: userauth.NewTokenManager(config.Config{
			AppName:       "datalk-test",
			JWTSecret:     "test-secret",
			JWTAccessTTL:  time.Minute,
			JWTRefreshTTL: time.Hour,
		}),
	}
}
