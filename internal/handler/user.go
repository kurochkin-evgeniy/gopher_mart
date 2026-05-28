// Package handler содержит HTTP-хендлеры бизнес-логики.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/kurochkin-evgeniy/gopher_mart/internal/auth"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/logging"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/storage/postgres"
	"golang.org/x/crypto/bcrypt"
)

// UserStorage описывает операции хранилища пользователей.
type UserStorage interface {
	// CreateUser создаёт пользователя и возвращает его идентификатор.
	CreateUser(ctx context.Context, login, passwordHash string) (int64, error)
}

// UserHandler реализует обработчики HTTP запросов, связанных с пользователями.
type UserHandler struct {
	storage UserStorage
}

// NewUserHandler создаёт хендлер пользователя на базе указанного хранилища.
func NewUserHandler(storage UserStorage) *UserHandler {
	return &UserHandler{storage: storage}
}

type credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Register обрабатывает регистрацию пользователя по адресу POST /api/user/register.
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var cred credentials
	if err := decodeJSON(r.Body, &cred); err != nil {
		logging.Sugar.Warnw("invalid register request", "error", err)
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	if cred.Login == "" || cred.Password == "" {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cred.Password), bcrypt.DefaultCost)
	if err != nil {
		logging.Sugar.Errorw("hash password", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	userID, err := h.storage.CreateUser(r.Context(), cred.Login, string(passwordHash))
	if err != nil {
		if errors.Is(err, postgres.ErrLoginTaken) {
			http.Error(w, "login already taken", http.StatusConflict)
			return
		}
		logging.Sugar.Errorw("create user", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(userID)
	if err != nil {
		logging.Sugar.Errorw("generate token", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set(auth.AuthHeader, auth.AuthScheme+" "+token)
	w.WriteHeader(http.StatusOK)
}

func decodeJSON(body io.Reader, v any) error {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
