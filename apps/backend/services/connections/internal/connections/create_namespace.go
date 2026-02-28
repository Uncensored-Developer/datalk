package connections

import (
	"context"
	"errors"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/mdobak/go-xerrors"
)

type NewNamespace struct {
	Name          string
	NamespaceType connections.NamespaceType
	ConnectionID  int32
}

func (n *NewNamespace) Validate() error {
	if n.Name == "" {
		return errors.New("name is required")
	}
	if n.NamespaceType == "" {
		return errors.New("namespace type is required")
	}
	switch n.NamespaceType {
	case connections.NamespaceTypeSchema, connections.NamespaceTypeDatabase, connections.NamespaceTypeKeyspace:
		// valid
	default:
		return errors.New("namespace type is invalid")
	}
	if n.ConnectionID <= 0 {
		return errors.New("connection id is required")
	}
	return nil
}

func (s *Service) CreateNamespace(ctx context.Context, newNamespace NewNamespace) (*connections.Namespace, error) {
	if err := newNamespace.Validate(); err != nil {
		return nil, err
	}

	namespace := connections.Namespace{
		Name:          newNamespace.Name,
		NamespaceType: newNamespace.NamespaceType,
		ConnectionID:  newNamespace.ConnectionID,
		IsEnabled:     true,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.storage.UpsertNamespace(ctx, &namespace); err != nil {
		return nil, xerrors.Newf("failed to insert namespace: %w", err)
	}

	return &namespace, nil
}
