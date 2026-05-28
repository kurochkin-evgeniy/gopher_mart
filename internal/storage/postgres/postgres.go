// Package postgres реализует доступ к данным в PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrLoginTaken возвращается, когда пытаются создать пользователя с уже занятым логином.
var ErrLoginTaken = errors.New("login already taken")

// Storage предоставляет методы для работы с данными пользователей в PostgreSQL.
type Storage struct {
	pool *pgxpool.Pool
}

// New создаёт хранилище и выполняет миграции схемы.
func New(ctx context.Context, databaseURI string) (*Storage, error) {
	if err := RunMigrations(databaseURI); err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(ctx, databaseURI)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Storage{pool: pool}, nil
}

// Close закрывает пул соединений к базе данных.
func (s *Storage) Close() {
	s.pool.Close()
}

// CreateUser создаёт пользователя с указанным логином и хешем пароля.
func (s *Storage) CreateUser(ctx context.Context, login, passwordHash string) (int64, error) {
	const query = `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id
	`

	var id int64
	err := s.pool.QueryRow(ctx, query, login, passwordHash).Scan(&id)
	if err != nil {
		return 0, mapCreateUserError(err)
	}

	return id, nil
}

// GetUserByLogin возвращает идентификатор пользователя и хеш пароля по логину.
func (s *Storage) GetUserByLogin(ctx context.Context, login string) (int64, string, error) {
	const query = `
		SELECT id, password_hash
		FROM users
		WHERE login = $1
	`

	var id int64
	var passwordHash string
	err := s.pool.QueryRow(ctx, query, login).Scan(&id, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", pgx.ErrNoRows
		}
		return 0, "", fmt.Errorf("get user by login: %w", err)
	}

	return id, passwordHash, nil
}

func mapCreateUserError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrLoginTaken
	}
	return fmt.Errorf("insert user: %w", err)
}
