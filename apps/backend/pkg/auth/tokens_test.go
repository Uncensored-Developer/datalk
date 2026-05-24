package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenManager_SignAndVerifyAccessToken(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	manager := NewTokenManager(config.Config{
		AppName:      "datalk-test",
		JWTSecret:    "test-secret",
		JWTAccessTTL: 2 * time.Minute,
		JWTIssuer:    "datalk-issuer",
	})

	token, expiresAt, err := manager.SignAccessToken(42, now)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, now.Add(2*time.Minute), expiresAt)

	userID, err := manager.VerifyAccessToken(token, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, int32(42), userID)
}

func TestTokenManager_Defaults(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	manager := NewTokenManager(config.Config{
		AppName:   "datalk-test",
		JWTSecret: "test-secret",
	})

	token, expiresAt, err := manager.SignAccessToken(7, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(15*time.Minute), expiresAt)
	assert.Equal(t, 30*24*time.Hour, manager.RefreshTTL())

	userID, err := manager.VerifyAccessToken(token, now)
	require.NoError(t, err)
	assert.Equal(t, int32(7), userID)
}

func TestTokenManager_UsesConfiguredIssuer(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	manager := NewTokenManager(config.Config{
		AppName:   "app-name",
		JWTSecret: "test-secret",
		JWTIssuer: "configured-issuer",
	})

	token, _, err := manager.SignAccessToken(7, now)
	require.NoError(t, err)

	claims := decodeClaims(t, token)
	assert.Equal(t, "configured-issuer", claims.Issuer)
}

func TestTokenManager_RequiresSecret(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager(config.Config{AppName: "datalk-test"})

	token, expiresAt, err := manager.SignAccessToken(7, time.Now().UTC())
	require.ErrorContains(t, err, "jwt secret is required")
	assert.Empty(t, token)
	assert.True(t, expiresAt.IsZero())

	userID, err := manager.VerifyAccessToken("a.b.c", time.Now().UTC())
	require.ErrorContains(t, err, "jwt secret is required")
	assert.Zero(t, userID)
}

func TestTokenManager_VerifyAccessTokenRejectsInvalidTokens(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	manager := NewTokenManager(config.Config{
		AppName:      "datalk-test",
		JWTSecret:    "test-secret",
		JWTAccessTTL: time.Minute,
	})
	token, _, err := manager.SignAccessToken(7, now)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		now     time.Time
		wantErr string
	}{
		{
			name:    "malformed",
			token:   "not-a-jwt",
			now:     now,
			wantErr: "invalid token",
		},
		{
			name:    "tampered signature",
			token:   token[:len(token)-1] + "x",
			now:     now,
			wantErr: "invalid token signature",
		},
		{
			name:    "expired",
			token:   token,
			now:     now.Add(time.Minute),
			wantErr: "token expired",
		},
		{
			name:    "wrong issuer",
			token:   tokenWithClaims(t, manager, token, func(claims *accessClaims) { claims.Issuer = "other" }),
			now:     now,
			wantErr: "invalid token issuer",
		},
		{
			name:    "wrong token type",
			token:   tokenWithClaims(t, manager, token, func(claims *accessClaims) { claims.TokenType = "refresh" }),
			now:     now,
			wantErr: "invalid token type",
		},
		{
			name:    "invalid subject",
			token:   tokenWithClaims(t, manager, token, func(claims *accessClaims) { claims.Subject = "0" }),
			now:     now,
			wantErr: "invalid token subject",
		},
		{
			name: "wrong algorithm",
			token: tokenWithHeader(t, manager, token, func(header map[string]string) {
				header["alg"] = "none"
			}),
			now:     now,
			wantErr: "invalid token algorithm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userID, err := manager.VerifyAccessToken(tt.token, tt.now)
			require.ErrorContains(t, err, tt.wantErr)
			assert.Zero(t, userID)
		})
	}
}

func TestTokenManager_NewRefreshToken(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager(config.Config{AppName: "datalk-test"})

	token, tokenHash, err := manager.NewRefreshToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, HashRefreshToken(token), tokenHash)
	assert.NotEqual(t, token, tokenHash)

	decoded, err := base64.RawURLEncoding.DecodeString(token)
	require.NoError(t, err)
	assert.Len(t, decoded, 32)
}

func TestHashRefreshToken(t *testing.T) {
	t.Parallel()

	hash := HashRefreshToken("refresh-token")
	assert.Equal(t, hash, HashRefreshToken("refresh-token"))
	assert.NotEqual(t, hash, HashRefreshToken("other-refresh-token"))

	decoded, err := base64.RawURLEncoding.DecodeString(hash)
	require.NoError(t, err)
	assert.Len(t, decoded, 32)
}

func decodeClaims(t *testing.T, token string) accessClaims {
	t.Helper()

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims accessClaims
	require.NoError(t, json.Unmarshal(claimsJSON, &claims))
	return claims
}

func tokenWithClaims(t *testing.T, manager *TokenManager, token string, mutate func(*accessClaims)) string {
	t.Helper()

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	claims := decodeClaims(t, token)
	mutate(&claims)

	claimsJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	unsigned := parts[0] + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	return unsigned + "." + manager.sign(unsigned)
}

func tokenWithHeader(t *testing.T, manager *TokenManager, token string, mutate func(map[string]string)) string {
	t.Helper()

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)

	var header map[string]string
	require.NoError(t, json.Unmarshal(headerJSON, &header))
	mutate(header)

	headerJSON, err = json.Marshal(header)
	require.NoError(t, err)

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + parts[1]
	return unsigned + "." + manager.sign(unsigned)
}
