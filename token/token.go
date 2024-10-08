package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
)

var (
	FailedToken = errors.New("missing authentication token")
)

type ParseToken struct {
	Role string
}

type Token struct {
	jwtSecretKey string
}

func NewToken(jwtSecretKey string) *Token {
	return &Token{
		jwtSecretKey: jwtSecretKey,
	}
}

func (t *Token) GenerateToken(role string) (string, error) {
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["role"] = role
	atClaims["exp"] = time.Now().Add(time.Minute * 10).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	token, err := at.SignedString([]byte(t.jwtSecretKey))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (t *Token) ParseToken(tokenStr string) (*ParseToken, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(t.jwtSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok && !token.Valid {
		return nil, FailedToken
	}

	return &ParseToken{
		Role: claims["role"].(string),
	}, nil
}
