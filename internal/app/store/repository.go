package store

import "billing_API/internal/app/model"

// UserRepository ...
type UserRepository interface {
	Create(*model.User) error
	Find(int) (*model.User, error)
	FindByEmail(string) (*model.User, error)
	Restore(userID int) error
}

// TransactionRepository handles transactions between users.
type TransactionRepository interface {
	Create(*model.Transaction) error
	List() ([]*model.Transaction, error)
	Cancel(transactionID int, userID int) error
	Confirm(transactionID int, userID int) error
}
