package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/kurochkin-evgeniy/gopher_mart/internal/model"
)

const balanceSQL = `
	SELECT
		COALESCE((
			SELECT SUM(accrual)
			FROM orders
			WHERE user_id = $1 AND status = 'PROCESSED' AND accrual IS NOT NULL
		), 0) - COALESCE((
			SELECT SUM(sum)
			FROM withdrawals
			WHERE user_id = $1
		), 0) AS current,
		COALESCE((
			SELECT SUM(sum)
			FROM withdrawals
			WHERE user_id = $1
		), 0) AS withdrawn
`

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func queryBalance(ctx context.Context, q rowQuerier, userID int64) (model.Balance, error) {
	var balance model.Balance
	if err := q.QueryRow(ctx, balanceSQL, userID).Scan(&balance.Current, &balance.Withdrawn); err != nil {
		return model.Balance{}, fmt.Errorf("query balance: %w", err)
	}
	return balance, nil
}

// GetBalance возвращает текущий баланс и сумму списаний пользователя.
func (s *Storage) GetBalance(ctx context.Context, userID int64) (model.Balance, error) {
	balance, err := queryBalance(ctx, s.pool, userID)
	if err != nil {
		return model.Balance{}, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

// Withdraw регистрирует списание баллов, если на счёте достаточно средств.
func (s *Storage) Withdraw(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	balance, err := queryBalance(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("get current balance: %w", err)
	}

	if balance.Current < sum {
		return model.ErrInsufficientFunds
	}

	const insertQuery = `
		INSERT INTO withdrawals (user_id, order_number, sum)
		VALUES ($1, $2, $3)
	`

	if _, err := tx.Exec(ctx, insertQuery, userID, orderNumber, sum); err != nil {
		return fmt.Errorf("insert withdrawal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit withdrawal: %w", err)
	}

	return nil
}

// ListWithdrawals возвращает списания пользователя, отсортированные по processed_at (новые первые).
func (s *Storage) ListWithdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	const query = `
		SELECT order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list withdrawals: %w", err)
	}
	defer rows.Close()

	var withdrawals []model.Withdrawal
	for rows.Next() {
		var w model.Withdrawal
		if err := rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, fmt.Errorf("scan withdrawal: %w", err)
		}
		withdrawals = append(withdrawals, w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate withdrawals: %w", err)
	}

	return withdrawals, nil
}

// ListOrdersForAccrual возвращает заказы, требующие опроса системы начислений.
func (s *Storage) ListOrdersForAccrual(ctx context.Context) ([]string, error) {
	const query = `
		SELECT number
		FROM orders
		WHERE status IN ('NEW', 'PROCESSING')
		ORDER BY uploaded_at
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list orders for accrual: %w", err)
	}
	defer rows.Close()

	var numbers []string
	for rows.Next() {
		var number string
		if err := rows.Scan(&number); err != nil {
			return nil, fmt.Errorf("scan order number: %w", err)
		}
		numbers = append(numbers, number)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders for accrual: %w", err)
	}

	return numbers, nil
}

// UpdateOrderAccrual обновляет статус и начисление заказа.
func (s *Storage) UpdateOrderAccrual(ctx context.Context, number, status string, accrual *float64) error {
	const query = `
		UPDATE orders
		SET status = $2, accrual = $3
		WHERE number = $1
	`

	tag, err := s.pool.Exec(ctx, query, number, status, accrual)
	if err != nil {
		return fmt.Errorf("update order accrual: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}

	return nil
}
