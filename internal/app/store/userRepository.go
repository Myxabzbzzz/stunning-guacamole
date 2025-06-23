package store

import (
	"database/sql"
	"dodobackend/cmd/model"
	"errors"
	"fmt"
)

type UserRepository struct {
	store *Store
}

func (r *UserRepository) Create(u *model.User) (model.User, error) {
	// Вставляем данные в таблицу users и получаем ID нового пользователя
	err := r.store.db.QueryRow(
		"INSERT INTO users (name, email, card_number, balance) VALUES ($1, $2, $3, $4) RETURNING id",
		u.Name,
		u.Email,
		u.CardNumber,
		u.Balance,
	).Scan(&u.ID)

	if err != nil {
		return model.User{}, fmt.Errorf("error inserting user: %w", err)
	}
	return *u, nil
}

func (r *UserRepository) FindByEmail(email string) (model.User, error) {

	var u model.User
	err := r.store.db.QueryRow(
		"SELECT id, name, email, card_number, balance FROM users WHERE email = $1",
		email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.CardNumber, &u.Balance)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, nil
		}
		return model.User{}, fmt.Errorf("error finding user by email: %w", err)
	}
	return u, nil
}
