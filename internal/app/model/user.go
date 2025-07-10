package model

import (
	"fmt"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"golang.org/x/crypto/bcrypt"
)

// User ...
type User struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	Password          string `json:"password,omitempty"`
	EncryptedPassword string `json:"-"`
	IsDeleted         bool   `json:"is_deleted"`
	AmountOfMoney     int64  `json:"amount_of_money"`
	PhoneNumber       string `json:"phone_number"`
	CardNumber        string `json:"card_number"`
}

// Validate ...
func (u *User) Validate() error {
	return validation.ValidateStruct(
		u,
		validation.Field(&u.Name, validation.Required.Error("name is required")),
		validation.Field(&u.PhoneNumber, validation.Required.Error("phone_number is required"), validation.Match(regexp.MustCompile(`^\+\d{11,15}$`)).Error("phone_number must be in format +12345678901")),
		validation.Field(&u.CardNumber, validation.Required.Error("card_number is required"), validation.Match(regexp.MustCompile(`^\d{16}$`)).Error("card_number must be 16 digits")),
		validation.Field(&u.AmountOfMoney, validation.Required.Error("amount_of_money is required"), validation.Min(0).Error("amount_of_money must be >= 0")),
		validation.Field(&u.Email, validation.Required.Error("email is required"), is.Email.Error("invalid email format")),
		validation.Field(&u.Password, validation.By(requiredIf(u.EncryptedPassword == "")), validation.Length(6, 100).Error("password must be at least 6 characters")),
	)
}

// BeforeCreate ...
func (u *User) BeforeCreate() error {
	if len(u.Password) > 0 {
		enc, err := encryptString(u.Password)
		if err != nil {
			return err
		}

		u.EncryptedPassword = enc
	}

	return nil
}

// Sanitize ...
func (u *User) Sanitize() {
	u.Password = ""
}

// ComparePassword ...
func (u *User) ComparePassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password)) == nil
}

// CheckNotDeleted
func (u *User) CheckNotDeleted() error {
	if u.IsDeleted {
		return fmt.Errorf("User is deleted or blocked")
	}
	return nil
}

func encryptString(s string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.MinCost)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
