package connections

import (
	"database/sql"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/secrets"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage/db"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/mdobak/go-xerrors"
)

type Service struct {
	*base.Base

	storage storage.Storage
	cipher  secrets.Cipher
}

func NewService(conn *sql.DB, cfg config.Config, logger *slog.Logger) *Service {
	cipher, err := secrets.NewAESCipher(cfg.ProviderConfigSecret)
	if err != nil {
		panic(err)
	}

	return &Service{
		Base:    base.NewBase("connections-core", logger, cfg),
		storage: db.NewStorage(conn),
		cipher:  cipher,
	}
}

func (s *Service) configuredCipher() secrets.Cipher {
	if s == nil || s.cipher == nil {
		return secrets.PlaintextCipher{}
	}
	return s.cipher
}

func (s *Service) encryptConnectionDSN(connection *connections.Connection) error {
	if connection == nil || connection.DSN == "" {
		return nil
	}

	encrypted, err := s.configuredCipher().Encrypt(connection.DSN)
	if err != nil {
		return xerrors.Newf("failed to encrypt connection dsn: %w", err)
	}
	connection.DSN = encrypted
	return nil
}

func (s *Service) decryptConnectionDSN(connection *connections.Connection) error {
	if connection == nil || connection.DSN == "" {
		return nil
	}

	plaintext, err := s.configuredCipher().Decrypt(connection.DSN)
	if err != nil {
		return xerrors.Newf("failed to decrypt connection dsn: %w", err)
	}
	connection.DSN = plaintext
	return nil
}

func (s *Service) decryptConnectionDSNs(connections []*connections.Connection) error {
	for _, connection := range connections {
		if err := s.decryptConnectionDSN(connection); err != nil {
			return err
		}
	}
	return nil
}
