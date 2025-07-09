package sqlstore

import (
	"billing_API/internal/app/model"
	"billing_API/internal/app/store"
	"database/sql"
	"errors"
)

// UserRepository ...
type UserRepository struct {
	store *Store
}

// Create ...
func (r *UserRepository) Create(u *model.User) error {
	if err := u.Validate(); err != nil {
		return err
	}
	if err := u.BeforeCreate(); err != nil {
		return err
	}
	// Проверка уникальности email
	var exists int
	r.store.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", u.Email).Scan(&exists)
	if exists > 0 {
		return errors.New("email already exists")
	}
	// Проверка уникальности phone_number
	r.store.db.QueryRow("SELECT COUNT(*) FROM users WHERE phone_number = $1", u.PhoneNumber).Scan(&exists)
	if exists > 0 {
		return errors.New("phone_number already exists")
	}
	// Проверка уникальности card_number
	r.store.db.QueryRow("SELECT COUNT(*) FROM users WHERE card_number = $1", u.CardNumber).Scan(&exists)
	if exists > 0 {
		return errors.New("card_number already exists")
	}
	return r.store.db.QueryRow(
		"INSERT INTO users (name, phone_number, card_number, email, encrypted_password, is_deleted, amount_of_money) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		u.Name,
		u.PhoneNumber,
		u.CardNumber,
		u.Email,
		u.EncryptedPassword,
		false,
		u.AmountOfMoney,
	).Scan(&u.ID)
}

// Find ...
func (r *UserRepository) Find(id int) (*model.User, error) {
	u := &model.User{}
	if err := r.store.db.QueryRow(
		"SELECT id, name, phone_number, card_number, email, encrypted_password, is_deleted, amount_of_money FROM users WHERE id = $1",
		id,
	).Scan(
		&u.ID,
		&u.Name,
		&u.PhoneNumber,
		&u.CardNumber,
		&u.Email,
		&u.EncryptedPassword,
		&u.IsDeleted,
		&u.AmountOfMoney,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}
		return nil, err
	}
	return u, nil
}

// FindByEmail ...
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	u := &model.User{}
	if err := r.store.db.QueryRow(
		"SELECT id, name, phone_number, card_number, email, encrypted_password, is_deleted, amount_of_money FROM users WHERE email = $1",
		email,
	).Scan(
		&u.ID,
		&u.Name,
		&u.PhoneNumber,
		&u.CardNumber,
		&u.Email,
		&u.EncryptedPassword,
		&u.IsDeleted,
		&u.AmountOfMoney,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) Restore(userID int) error {
	u := &model.User{}
	err := r.store.db.QueryRow("SELECT id, is_deleted FROM users WHERE id = $1", userID).Scan(&u.ID, &u.IsDeleted)
	if err != nil {
		return errors.New("user not found")
	}
	if !u.IsDeleted {
		return errors.New("user is not deleted")
	}
	_, err = r.store.db.Exec("UPDATE users SET is_deleted = FALSE WHERE id = $1", userID)
	return err
}
