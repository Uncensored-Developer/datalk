package hashers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArgon2Hasher(t *testing.T) {
	t.Parallel()

	password := "test_password"

	argon2Hasher := NewArgon2Hasher()
	hash, err := argon2Hasher.Hash(t.Context(), password)
	require.NoError(t, err)

	got, err := argon2Hasher.Verify(t.Context(), password, hash)
	require.NoError(t, err)
	require.True(t, got)
}
