package sqlstore

import (
	"billing_API/internal/app/model"
	"billing_API/internal/app/store"
	"database/sql"
	"errors"
)

// Store ...
type Store struct {
	db                    *sql.DB
	userRepository        *UserRepository
	transactionRepository *TransactionRepository
}

// New ...
func New(db *sql.DB) *Store {
	return &Store{
		db: db,
	}
}

// User ...
func (s *Store) User() store.UserRepository {
	if s.userRepository != nil {
		return s.userRepository
	}

	s.userRepository = &UserRepository{
		store: s,
	}

	return s.userRepository
}

// Transaction ...
func (s *Store) Transaction() store.TransactionRepository {
	if s.transactionRepository != nil {
		return s.transactionRepository
	}

	s.transactionRepository = &TransactionRepository{
		store: s,
	}

	return s.transactionRepository
}

// TransactionRepository handles DB operations for transactions.
type TransactionRepository struct {
	store *Store
}

func (r *TransactionRepository) Create(t *model.Transaction) error {
	if t.AmountOfMoney <= 0 {
		return errors.New("amount_of_money должен быть положительным")
	}
	if t.FromUserID == t.ToUserID {
		return errors.New("отправитель и получатель не могут совпадать")
	}
	// Проверяем, что оба пользователя существуют и не удалены
	var fromDeleted, toDeleted bool
	err := r.store.db.QueryRow("SELECT is_deleted FROM users WHERE id = $1", t.FromUserID).Scan(&fromDeleted)
	if err != nil {
		return errors.New("отправитель не найден или удалён")
	}
	if fromDeleted {
		return errors.New("отправитель удалён")
	}
	err = r.store.db.QueryRow("SELECT is_deleted FROM users WHERE id = $1", t.ToUserID).Scan(&toDeleted)
	if err != nil {
		return errors.New("получатель не найден или удалён")
	}
	if toDeleted {
		return errors.New("получатель удалён")
	}
	return r.store.db.QueryRow(
		"INSERT INTO transactions (from_user_id, to_user_id, amount_of_money, status) VALUES ($1, $2, $3, $4) RETURNING id, transaction_time, is_deleted, status",
		t.FromUserID,
		t.ToUserID,
		t.AmountOfMoney,
		model.TransactionStatusCreated,
	).Scan(&t.ID, &t.TransactionTime, &t.IsDeleted, &t.Status)
}

func (r *TransactionRepository) List() ([]*model.Transaction, error) {
	rows, err := r.store.db.Query("SELECT id, from_user_id, to_user_id, amount_of_money, transaction_time, status, is_deleted FROM transactions ORDER BY transaction_time DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*model.Transaction
	for rows.Next() {
		t := &model.Transaction{}
		if err := rows.Scan(&t.ID, &t.FromUserID, &t.ToUserID, &t.AmountOfMoney, &t.TransactionTime, &t.Status, &t.IsDeleted); err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

// Confirm sets status to 'confirmed' and transfers money from sender to receiver.
func (r *TransactionRepository) Confirm(transactionID int, userID int) error {
	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var fromUserID int
	var status string
	var amount int64
	err = tx.QueryRow("SELECT from_user_id, status, amount_of_money FROM transactions WHERE id = $1 FOR UPDATE", transactionID).Scan(&fromUserID, &status, &amount)
	if err != nil {
		return err
	}
	if fromUserID != userID {
		return errors.New("только отправитель может подтверждать транзакцию")
	}
	if status != model.TransactionStatusCreated && status != model.TransactionStatusPending {
		return errors.New("транзакция уже подтверждена или отменена")
	}
	// Проверить баланс отправителя
	var senderBalance int64
	var senderDeleted, recipientDeleted bool
	err = tx.QueryRow("SELECT amount_of_money, is_deleted FROM users WHERE id = $1 FOR UPDATE", fromUserID).Scan(&senderBalance, &senderDeleted)
	if err != nil || senderDeleted {
		return errors.New("отправитель не найден или удалён")
	}
	var toUserID int
	err = tx.QueryRow("SELECT to_user_id FROM transactions WHERE id = $1", transactionID).Scan(&toUserID)
	if err != nil {
		return err
	}
	err = tx.QueryRow("SELECT is_deleted FROM users WHERE id = $1 FOR UPDATE", toUserID).Scan(&recipientDeleted)
	if err != nil || recipientDeleted {
		return errors.New("получатель не найден или удалён")
	}
	if senderBalance < amount {
		return errors.New("недостаточно средств у отправителя")
	}
	_, err = tx.Exec("UPDATE users SET amount_of_money = amount_of_money - $1 WHERE id = $2", amount, fromUserID)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE users SET amount_of_money = amount_of_money + $1 WHERE id = $2", amount, toUserID)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE transactions SET status = $1 WHERE id = $2", model.TransactionStatusConfirmed, transactionID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Cancel sets status to 'canceled' and returns the money to the sender if possible.
func (r *TransactionRepository) Cancel(transactionID int, userID int) error {
	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var fromUserID, toUserID int
	var amount int64
	var status string
	err = tx.QueryRow("SELECT from_user_id, to_user_id, amount_of_money, status FROM transactions WHERE id = $1 FOR UPDATE", transactionID).Scan(&fromUserID, &toUserID, &amount, &status)
	if err != nil {
		return err
	}
	if fromUserID != userID {
		return errors.New("только отправитель может отменять транзакцию")
	}
	if status == model.TransactionStatusCanceled || status == model.TransactionStatusConfirmed {
		return nil // Already canceled or confirmed
	}
	// Set status to canceled
	_, err = tx.Exec("UPDATE transactions SET status = $1 WHERE id = $2", model.TransactionStatusCanceled, transactionID)
	if err != nil {
		return err
	}
	// Return the money to the sender
	_, err = tx.Exec("UPDATE users SET amount_of_money = amount_of_money + $1 WHERE id = $2", amount, fromUserID)
	if err != nil {
		return err
	}
	// Subtract the money from the receiver, но не допускать отрицательного баланса
	var receiverBalance int64
	err = tx.QueryRow("SELECT amount_of_money FROM users WHERE id = $1 FOR UPDATE", toUserID).Scan(&receiverBalance)
	if err != nil {
		return err
	}
	if receiverBalance < amount {
		return errors.New("у получателя недостаточно средств для возврата")
	}
	_, err = tx.Exec("UPDATE users SET amount_of_money = amount_of_money - $1 WHERE id = $2", amount, toUserID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DB() *sql.DB {
	return s.db
}
