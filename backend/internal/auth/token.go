package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid access token")

type Tokens struct{ secret []byte }

func NewTokens(secret string) *Tokens { return &Tokens{secret: []byte(secret)} }

func (t *Tokens) Issue(userID uint64) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    "devhub",
		Subject:   strconv.FormatUint(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(12 * time.Hour)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

func (t *Tokens) Verify(raw string) (uint64, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("devhub"),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return 0, ErrInvalidToken
	}
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrInvalidToken
	}
	return id, nil
}
