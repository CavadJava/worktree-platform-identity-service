package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type Claims struct {
	UserID        string  `json:"user_id"`
	Role          string  `json:"role"`
	ShopID        *string `json:"shop_id,omitempty"`
	ShopRoleLevel int     `json:"shop_role_level"`
	jwt.RegisteredClaims
}

// JWTManager verifies tokens issued by registration-service. Both services
// must share the same JWT_SECRET for this to work.
type JWTManager struct {
	secret []byte
}

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{secret: []byte(secret)}
}

func (m *JWTManager) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
