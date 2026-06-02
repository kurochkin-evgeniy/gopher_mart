// Package model содержит доменные типы приложения.
package model

import (
	"errors"
	"time"
)

// ErrInsufficientFunds возвращается, когда на счёте недостаточно баллов для списания.
var ErrInsufficientFunds = errors.New("insufficient funds")

// Withdrawal описывает операцию списания баллов.
type Withdrawal struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

// Balance содержит текущий баланс и сумму всех списаний пользователя.
type Balance struct {
	Current   float64
	Withdrawn float64
}
