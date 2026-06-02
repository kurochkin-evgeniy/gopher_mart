package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrOrderNotFound возвращается, когда заказ с указанным номером не найден.
var ErrOrderNotFound = errors.New("order not found")

// ErrOrderTaken возвращается при попытке загрузить номер, уже принадлежащий другому пользователю.
var ErrOrderTaken = errors.New("order already taken")

// Order описывает заказ пользователя в системе лояльности.
type Order struct {
	Number     string
	Status     string
	Accrual    *float64
	UploadedAt time.Time
}

// GetOrderByNumber возвращает заказ по номеру.
func (s *Storage) GetOrderByNumber(ctx context.Context, number string) (Order, int64, error) {
	const query = `
		SELECT user_id, status, accrual, uploaded_at
		FROM orders
		WHERE number = $1
	`

	var order Order
	var userID int64
	order.Number = number

	err := s.pool.QueryRow(ctx, query, number).Scan(&userID, &order.Status, &order.Accrual, &order.UploadedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, 0, ErrOrderNotFound
		}
		return Order{}, 0, fmt.Errorf("get order by number: %w", err)
	}

	return order, userID, nil
}

// CreateOrder создаёт новый заказ со статусом NEW.
func (s *Storage) CreateOrder(ctx context.Context, userID int64, number string) error {
	const query = `
		INSERT INTO orders (user_id, number, status)
		VALUES ($1, $2, 'NEW')
	`

	_, err := s.pool.Exec(ctx, query, userID, number)
	if err != nil {
		return mapCreateOrderError(err)
	}

	return nil
}

// ListOrdersByUser возвращает заказы пользователя, отсортированные по uploaded_at (новые первые).
func (s *Storage) ListOrdersByUser(ctx context.Context, userID int64) ([]Order, error) {
	const query = `
		SELECT number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}

	return orders, nil
}

func mapCreateOrderError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrOrderTaken
	}
	return fmt.Errorf("insert order: %w", err)
}
