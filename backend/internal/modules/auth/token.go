package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenUse  = "access"
	refreshTokenUse = "refresh"
)

type Claims struct {
	TokenUse string `json:"token_use"`
	Role     string `json:"role,omitempty"`
	Email    string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewTokenManager(secret, issuer string, accessTTL, refreshTTL time.Duration) (*TokenManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must contain at least 32 characters")
	}
	if accessTTL <= 0 || refreshTTL <= 0 || refreshTTL <= accessTTL {
		return nil, errors.New("token TTL values are invalid")
	}

	return &TokenManager{
		secret:     []byte(secret),
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}, nil
}

func (m *TokenManager) IssuePair(user User) (TokenPair, error) {
	now := m.now().UTC()
	accessExpiresAt := now.Add(m.accessTTL)
	refreshExpiresAt := now.Add(m.refreshTTL)

	accessID, err := randomTokenID()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate access token id: %w", err)
	}
	refreshID, err := randomTokenID()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token id: %w", err)
	}

	accessToken, err := m.sign(Claims{
		TokenUse: accessTokenUse,
		Role:     user.Role,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   user.ID,
			ID:        accessID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
		},
	})
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := m.sign(Claims{
		TokenUse: refreshTokenUse,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   user.ID,
			ID:        refreshID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
		},
	})
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresInSeconds: int64(m.accessTTL.Seconds()),
		refreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (m *TokenManager) ParseAccess(rawToken string) (Claims, error) {
	return m.parse(rawToken, accessTokenUse)
}

func (m *TokenManager) ParseRefresh(rawToken string) (Claims, error) {
	return m.parse(rawToken, refreshTokenUse)
}

func (m *TokenManager) sign(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (m *TokenManager) parse(rawToken, expectedUse string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(
		rawToken,
		&claims,
		func(token *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
	)
	if err != nil || !token.Valid || claims.TokenUse != expectedUse || claims.Subject == "" {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

func HashToken(rawToken string) string {
	digest := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(digest[:])
}

func randomTokenID() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
