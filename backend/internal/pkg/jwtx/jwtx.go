// Package jwtx issues and verifies HS256 tokens with an audience per client type.
package jwtx

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Role string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

type Signer struct {
	secret []byte
	aud    string
	ttl    time.Duration
}

func New(secret, aud string, ttl time.Duration) *Signer {
	return &Signer{secret: []byte(secret), aud: aud, ttl: ttl}
}

func (s *Signer) Sign(sub, role string) (string, time.Time, error) {
	exp := time.Now().Add(s.ttl)
	c := Claims{Role: role, RegisteredClaims: jwt.RegisteredClaims{
		Subject:   sub,
		Audience:  jwt.ClaimStrings{s.aud},
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.secret)
	return tok, exp, err
}

func (s *Signer) Parse(token string) (*Claims, error) {
	var c Claims
	t, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithAudience(s.aud))
	if err != nil || !t.Valid {
		return nil, errors.New("invalid token")
	}
	return &c, nil
}

func (s *Signer) TTL() time.Duration { return s.ttl }
