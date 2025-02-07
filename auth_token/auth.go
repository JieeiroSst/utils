package auth_token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type TokenManager struct {
	accessTokenSecret  []byte
	refreshTokenSecret []byte
	accessTokenTTL     time.Duration
	refreshTokenTTL    time.Duration
}

type TokenClaims struct {
	jwt.StandardClaims
	UserID    string `json:"user_id"`
	TokenID   string `json:"token_id"`
	TokenType string `json:"token_type"`
}

func NewTokenManager(accessSecret, refreshSecret []byte, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		accessTokenSecret:  accessSecret,
		refreshTokenSecret: refreshSecret,
		accessTokenTTL:     accessTTL,
		refreshTokenTTL:    refreshTTL,
	}
}

func generateTokenID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (tm *TokenManager) CreateTokenPair(userID string) (accessToken, refreshToken string, err error) {
	accessTokenID, err := generateTokenID()
	if err != nil {
		return "", "", err
	}

	refreshTokenID, err := generateTokenID()
	if err != nil {
		return "", "", err
	}

	accessTokenClaims := TokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(tm.accessTokenTTL).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		UserID:    userID,
		TokenID:   accessTokenID,
		TokenType: "access",
	}

	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims).
		SignedString(tm.accessTokenSecret)
	if err != nil {
		return "", "", err
	}

	refreshTokenClaims := TokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(tm.refreshTokenTTL).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		UserID:    userID,
		TokenID:   refreshTokenID,
		TokenType: "refresh",
	}

	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims).
		SignedString(tm.refreshTokenSecret)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (tm *TokenManager) ValidateToken(tokenString string, isRefresh bool) (*TokenClaims, error) {
	var secret []byte
	if isRefresh {
		secret = tm.refreshTokenSecret
	} else {
		secret = tm.accessTokenSecret
	}

	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	expectedType := "access"
	if isRefresh {
		expectedType = "refresh"
	}
	if claims.TokenType != expectedType {
		return nil, errors.New("invalid token type")
	}

	return claims, nil
}

func (tm *TokenManager) RefreshTokens(refreshToken string) (newAccessToken, newRefreshToken string, err error) {
	claims, err := tm.ValidateToken(refreshToken, true)
	if err != nil {
		return "", "", err
	}

	return tm.CreateTokenPair(claims.UserID)
}
