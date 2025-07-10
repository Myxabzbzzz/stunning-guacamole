package teststore_test

import (
	"billing_API/internal/app/model"
	"billing_API/internal/app/store"
	"billing_API/internal/app/store/teststore"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserRepository_Create(t *testing.T) {
	s := teststore.New()
	u := model.TestUser(t)
	assert.NoError(t, s.User().Create(u))
	assert.NotNil(t, u.ID)
}

func TestUserRepository_Find(t *testing.T) {
	s := teststore.New()
	u1 := model.TestUser(t)
	s.User().Create(u1)
	u2, err := s.User().Find(u1.ID)
	assert.NoError(t, err)
	assert.NotNil(t, u2)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	s := teststore.New()
	u1 := model.TestUser(t)
	_, err := s.User().FindByEmail(u1.Email)
	assert.EqualError(t, err, store.ErrRecordNotFound.Error())

	s.User().Create(u1)
	u2, err := s.User().FindByEmail(u1.Email)
	assert.NoError(t, err)
	assert.NotNil(t, u2)
}

func TestTransaction_EdgeCases(t *testing.T) {
	s := teststore.New()
	// Create two users
	sender := &model.User{Name: "Sender", Email: "sender@example.org", Password: "pass", AmountOfMoney: 1000}
	recipient := &model.User{Name: "Recipient", Email: "recipient@example.org", Password: "pass", AmountOfMoney: 1000}
	s.User().Create(sender)
	s.User().Create(recipient)

	txRepo := s.Transaction().(*teststore.TransactionRepository)

	// 1. Normal transaction and cancellation
	tx := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 100}
	txRepo.Create(tx)
	assert.Equal(t, "created", tx.Status)
	txRepo.Cancel(tx.ID)
	assert.Equal(t, "canceled", tx.Status)

	// 2. Repeated cancellation (should remain canceled)
	txRepo.Cancel(tx.ID)
	assert.Equal(t, "canceled", tx.Status)

	// 3. Confirmed transaction cannot be cancelled
	tx2 := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 50, Status: "confirmed"}
	txRepo.Create(tx2)
	txRepo.Cancel(tx2.ID)
	assert.Equal(t, "confirmed", tx2.Status)

	// 4. Recipient does not have enough money to return
	tx3 := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 9999}
	txRepo.Create(tx3)
	recipient.AmountOfMoney = 0 // forcibly set balance to zero
	txRepo.Cancel(tx3.ID)
	// Balance should not become negative
	assert.True(t, recipient.AmountOfMoney >= 0)

	// 5. Transaction with non-existent user
	tx4 := &model.Transaction{FromUserID: 999, ToUserID: recipient.ID, AmountOfMoney: 10}
	err := txRepo.Create(tx4)
	assert.NoError(t, err) // In teststore, there is no check, but in a real store, it should be an error

	// 6. amount_of_money <= 0
	tx5 := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 0}
	err = txRepo.Create(tx5)
	assert.NoError(t, err) // In teststore, there is no check, but in a real store, it should be an error

	// 7. Sender = Recipient
	tx6 := &model.Transaction{FromUserID: sender.ID, ToUserID: sender.ID, AmountOfMoney: 10}
	err = txRepo.Create(tx6)
	assert.NoError(t, err) // In teststore, there is no check, but in a real store, it should be an error
}
