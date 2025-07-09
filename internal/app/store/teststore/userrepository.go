package teststore

import (
	"billing_API/internal/app/model"
	"billing_API/internal/app/store"
	"errors"
)

// UserRepository ...
type UserRepository struct {
	store *Store
	Users map[int]*model.User
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
	for _, user := range r.Users {
		if user.Email == u.Email {
			return errors.New("email already exists")
		}
		if user.PhoneNumber == u.PhoneNumber {
			return errors.New("phone_number already exists")
		}
		if user.CardNumber == u.CardNumber {
			return errors.New("card_number already exists")
		}
	}
	u.ID = len(r.Users) + 1
	r.Users[u.ID] = u

	return nil
}

// Find ...
func (r *UserRepository) Find(id int) (*model.User, error) {
	u, ok := r.Users[id]
	if !ok {
		return nil, store.ErrRecordNotFound
	}

	return u, nil
}

// FindByEmail ...
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	for _, u := range r.Users {
		if u.Email == email {
			return u, nil
		}
	}

	return nil, store.ErrRecordNotFound
}

func (r *UserRepository) Restore(userID int) error {
	u, ok := r.Users[userID]
	if !ok {
		return errors.New("user not found")
	}
	if !u.IsDeleted {
		return errors.New("user is not deleted")
	}
	u.IsDeleted = false
	return nil
}
