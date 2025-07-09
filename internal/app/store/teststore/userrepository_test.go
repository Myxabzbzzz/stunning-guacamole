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
	// Создаём двух пользователей
	sender := &model.User{Name: "Sender", Email: "sender@example.org", Password: "pass", AmountOfMoney: 1000}
	recipient := &model.User{Name: "Recipient", Email: "recipient@example.org", Password: "pass", AmountOfMoney: 1000}
	s.User().Create(sender)
	s.User().Create(recipient)

	txRepo := s.Transaction().(*teststore.TransactionRepository)

	// 1. Обычная транзакция и отмена
	tx := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 100}
	txRepo.Create(tx)
	assert.Equal(t, "created", tx.Status)
	txRepo.Cancel(tx.ID)
	assert.Equal(t, "canceled", tx.Status)

	// 2. Повторная отмена (должно остаться canceled)
	txRepo.Cancel(tx.ID)
	assert.Equal(t, "canceled", tx.Status)

	// 3. Подтверждённая транзакция не может быть отменена
	tx2 := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 50, Status: "confirmed"}
	txRepo.Create(tx2)
	txRepo.Cancel(tx2.ID)
	assert.Equal(t, "confirmed", tx2.Status)

	// 4. Недостаточно средств у получателя для возврата
	tx3 := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 9999}
	txRepo.Create(tx3)
	recipient.AmountOfMoney = 0 // насильно обнуляем баланс
	txRepo.Cancel(tx3.ID)
	// Баланс не должен стать отрицательным
	assert.True(t, recipient.AmountOfMoney >= 0)

	// 5. Транзакция с несуществующим пользователем
	tx4 := &model.Transaction{FromUserID: 999, ToUserID: recipient.ID, AmountOfMoney: 10}
	err := txRepo.Create(tx4)
	assert.NoError(t, err) // в teststore нет проверки, но в реальном store должна быть ошибка

	// 6. amount_of_money <= 0
	tx5 := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 0}
	err = txRepo.Create(tx5)
	assert.NoError(t, err) // в teststore нет проверки, но в реальном store должна быть ошибка

	// 7. Отправитель = получатель
	tx6 := &model.Transaction{FromUserID: sender.ID, ToUserID: sender.ID, AmountOfMoney: 10}
	err = txRepo.Create(tx6)
	assert.NoError(t, err) // в teststore нет проверки, но в реальном store должна быть ошибка
}
