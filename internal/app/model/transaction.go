package model

import "time"

const (
	TransactionStatusPending   = "pending"
	TransactionStatusCreated   = "created"
	TransactionStatusConfirmed = "confirmed"
	TransactionStatusCanceled  = "canceled"
)

// Transaction represents a money transfer between two users.
type Transaction struct {
	ID              int       `json:"id"`
	FromUserID      int       `json:"from_user_id"`
	ToUserID        int       `json:"to_user_id"`
	AmountOfMoney   int64     `json:"amount_of_money"` // Stored as int64 (in cents)
	TransactionTime time.Time `json:"transaction_time"`
	Status          string    `json:"status"` // pending, created, confirmed, canceled
	IsDeleted       bool      `json:"is_deleted"`
}
