package secrets

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESCipher_RoundTrip(t *testing.T) {
	t.Parallel()

	cipher, err := NewAESCipher("test-secret")
	require.NoError(t, err)

	ciphertext, err := cipher.Encrypt("api-key")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(ciphertext, "v1:"))
	assert.NotContains(t, ciphertext, "api-key")

	plaintext, err := cipher.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "api-key", plaintext)
}

func TestAESCipher_RejectsEmptySecret(t *testing.T) {
	t.Parallel()

	_, err := NewAESCipher("   ")
	require.EqualError(t, err, "provider config secret is required")
}

func TestAESCipher_DecryptAllowsLegacyUnversionedValue(t *testing.T) {
	t.Parallel()

	cipher, err := NewAESCipher("test-secret")
	require.NoError(t, err)

	plaintext, err := cipher.Decrypt("legacy-api-key")
	require.NoError(t, err)
	assert.Equal(t, "legacy-api-key", plaintext)
}

func TestAESCipher_DecryptWithWrongSecretFails(t *testing.T) {
	t.Parallel()

	cipher, err := NewAESCipher("test-secret")
	require.NoError(t, err)
	wrongCipher, err := NewAESCipher("wrong-secret")
	require.NoError(t, err)

	ciphertext, err := cipher.Encrypt("api-key")
	require.NoError(t, err)

	_, err = wrongCipher.Decrypt(ciphertext)
	require.Error(t, err)
}
