package teststore

import (
	"billing_API/internal/app/model"
	"billing_API/internal/app/store"
	"errors"
)

// Store ...
type Store struct {
	userRepository        *UserRepository
	transactionRepository *TransactionRepository
}

type TransactionRepository struct {
	store        *Store
	transactions map[int]*model.Transaction
}

func (r *TransactionRepository) Create(t *model.Transaction) error {
	t.ID = len(r.transactions) + 1
	t.Status = "created"
	r.transactions[t.ID] = t
	return nil
}

func (r *TransactionRepository) List() ([]*model.Transaction, error) {
	var txs []*model.Transaction
	for _, tx := range r.transactions {
		txs = append(txs, tx)
	}
	return txs, nil
}

func (r *TransactionRepository) Confirm(transactionID int, userID int) error {
	tx, ok := r.transactions[transactionID]
	if !ok {
		return nil
	}
	if tx.FromUserID != userID {
		return errors.New("only the sender can confirm the transaction")
	}
	if tx.Status == "confirmed" || tx.Status == "canceled" {
		return errors.New("transaction already confirmed or canceled")
	}
	tx.Status = "confirmed"
	fromUser := r.store.userRepository.Users[tx.FromUserID]
	toUser := r.store.userRepository.Users[tx.ToUserID]
	if fromUser != nil && toUser != nil && fromUser.AmountOfMoney >= tx.AmountOfMoney {
		fromUser.AmountOfMoney -= tx.AmountOfMoney
		toUser.AmountOfMoney += tx.AmountOfMoney
		return nil
	}
	return errors.New("insufficient funds with the sender or recipient not found")
}

func (r *TransactionRepository) Cancel(transactionID int, userID int) error {
	tx, ok := r.transactions[transactionID]
	if !ok {
		return nil // already canceled or not found
	}
	if tx.FromUserID != userID {
		return errors.New("only the sender can cancel the transaction")
	}
	if tx.Status == "canceled" || tx.Status == "confirmed" {
		return nil
	}
	tx.Status = "canceled"
	fromUser := r.store.userRepository.Users[tx.FromUserID]
	toUser := r.store.userRepository.Users[tx.ToUserID]
	if fromUser != nil && toUser != nil && toUser.AmountOfMoney >= tx.AmountOfMoney {
		fromUser.AmountOfMoney += tx.AmountOfMoney
		toUser.AmountOfMoney -= tx.AmountOfMoney
		return nil
	}
	return errors.New("insufficient funds with the recipient for return or user not found")
}

func (s *Store) Transaction() store.TransactionRepository {
	if s.transactionRepository != nil {
		return s.transactionRepository
	}
	s.transactionRepository = &TransactionRepository{
		store:        s,
		transactions: make(map[int]*model.Transaction),
	}
	return s.transactionRepository
}

// New ...
func New() *Store {
	return &Store{}
}

// User ...
func (s *Store) User() store.UserRepository {
	if s.userRepository != nil {
		return s.userRepository
	}

	s.userRepository = &UserRepository{
		store: s,
		users: make(map[int]*model.User),
	}

	return s.userRepository
}
