// Package auth предоставляет функции для генерации и проверки JWT-токенов.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthHeader задаёт имя HTTP-заголовка, в котором передается токен.
const (
	AuthHeader = "Authorization"
	// AuthScheme задаёт схему в заголовке Authorization (например, "Bearer <token>").
	AuthScheme = "Bearer"
)

var (
	// ErrInvalidToken возвращается при попытке разобрать некорректный или просроченный токен.
	ErrInvalidToken = errors.New("invalid token")
	secret          []byte
)

// Init задаёт ключ подписи JWT. Вызывайте один раз при старте приложения (например, из main после загрузки конфигурации).
func Init(jwtSecret string) {
	secret = []byte(jwtSecret)
}

// Claims описывает содержимое JWT и идентификатор пользователя.
type Claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

// GenerateToken генерирует JWT для пользователя с идентификатором userID.
func GenerateToken(userID int64) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken разбирает строку токена и возвращает Claims при успешной валидации.
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
