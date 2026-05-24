package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

type RefreshToken struct {
	ID        int32
	UserID    int32
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

const jwtAlgorithm = "HS256"

type TokenPair struct {
	AccessToken        string
	RefreshToken       string
	ExpiresAt          time.Time
	MustChangePassword bool
}

type Session struct {
	User   *users.User
	Tokens TokenPair
}

type TokenManager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type accessClaims struct {
	Subject   string `json:"sub"`
	Issuer    string `json:"iss,omitempty"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	TokenType string `json:"typ"`
}

func NewTokenManager(cfg config.Config) *TokenManager {
	issuer := cfg.JWTIssuer
	if issuer == "" {
		issuer = cfg.AppName
	}
	accessTTL := cfg.JWTAccessTTL
	if accessTTL == 0 {
		accessTTL = 15 * time.Minute
	}
	refreshTTL := cfg.JWTRefreshTTL
	if refreshTTL == 0 {
		refreshTTL = 30 * 24 * time.Hour
	}
	return &TokenManager{
		secret:     []byte(cfg.JWTSecret),
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (m *TokenManager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

func (m *TokenManager) SignAccessToken(userID int32, now time.Time) (string, time.Time, error) {
	if len(m.secret) == 0 {
		return "", time.Time{}, xerrors.New("jwt secret is required")
	}

	expiresAt := now.Add(m.accessTTL)
	header := map[string]string{
		"alg": jwtAlgorithm,
		"typ": "JWT",
	}
	claims := accessClaims{
		Subject:   strconv.Itoa(int(userID)),
		Issuer:    m.issuer,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  now.Unix(),
		TokenType: "access",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", time.Time{}, err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := m.sign(unsigned)
	return unsigned + "." + signature, expiresAt, nil
}

func (m *TokenManager) VerifyAccessToken(token string, now time.Time) (int32, error) {
	if len(m.secret) == 0 {
		return 0, xerrors.New("jwt secret is required")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, xerrors.New("invalid token")
	}

	unsigned := parts[0] + "." + parts[1]
	expected := m.sign(unsigned)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return 0, xerrors.New("invalid token signature")
	}

	var header struct {
		Algorithm string `json:"alg"`
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, err
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return 0, err
	}
	if header.Algorithm != jwtAlgorithm {
		return 0, xerrors.New("invalid token algorithm")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, err
	}

	var claims accessClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return 0, err
	}
	if claims.TokenType != "access" {
		return 0, xerrors.New("invalid token type")
	}
	if claims.Issuer != m.issuer {
		return 0, xerrors.New("invalid token issuer")
	}
	if !now.Before(time.Unix(claims.ExpiresAt, 0)) {
		return 0, xerrors.New("token expired")
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 32)
	if err != nil || userID <= 0 {
		return 0, xerrors.New("invalid token subject")
	}
	return int32(userID), nil
}

func (m *TokenManager) NewRefreshToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", xerrors.Newf("generate refresh token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, HashRefreshToken(token), nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (m *TokenManager) sign(unsigned string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
