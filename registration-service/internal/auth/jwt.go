package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID        string  `json:"user_id"`
	Role          string  `json:"role"`
	ShopID        *string `json:"shop_id,omitempty"`
	ShopRoleLevel int     `json:"shop_role_level"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTManager(secret string, ttlMinutes int) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		ttl:    time.Duration(ttlMinutes) * time.Minute,
	}
}

func (m *JWTManager) Generate(userID, role string, shopID *string, shopRoleLevel int) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.ttl)
	claims := Claims{
		UserID:        userID,
		Role:          role,
		ShopID:        shopID,
		ShopRoleLevel: shopRoleLevel,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}
