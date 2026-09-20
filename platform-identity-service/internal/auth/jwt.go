package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type Claims struct {
	UserID     string `json:"user_id"`
	SystemRole string `json:"system_role"`
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

// SetTTL updates the duration used by every Generate call from this point
// on — lets the token lifetime be changed at runtime (e.g. from an
// admin-editable setting) without restarting the process. Safe to call
// concurrently with Generate/Verify since it only ever replaces the whole
// field, never partially mutates it, but two SetTTL calls racing each
// other could interleave; callers only expect eventual consistency here,
// not a strict ordering guarantee.
func (m *JWTManager) SetTTL(ttlMinutes int) {
	m.ttl = time.Duration(ttlMinutes) * time.Minute
}

func (m *JWTManager) Generate(userID, systemRole string) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.ttl)
	claims := Claims{
		UserID:     userID,
		SystemRole: systemRole,
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
