// Package middleware содержит HTTP-middleware приложения.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/auth"
)

type contextKey string

// UserIDKey — ключ контекста для идентификатора аутентифицированного пользователя.
const UserIDKey contextKey = "userID"

// Authenticate проверяет JWT из заголовка Authorization и сохраняет user ID в контексте.
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromRequest(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext возвращает идентификатор пользователя из контекста запроса.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}

func userIDFromRequest(r *http.Request) (int64, bool) {
	header := r.Header.Get(auth.AuthHeader)
	if header == "" {
		return 0, false
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != auth.AuthScheme {
		return 0, false
	}

	claims, err := auth.ParseToken(parts[1])
	if err != nil {
		return 0, false
	}

	return claims.UserID, true
}
