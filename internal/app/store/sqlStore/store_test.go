package sqlstore_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"billing_API/internal/app/model"
	"billing_API/internal/app/store/sqlstore"
)

var (
	databaseURL string
)

func TestMain(m *testing.M) {
	databaseURL = os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "host=localhost dbname=restapi_test sslmode=disable"
	}

	os.Exit(m.Run())
}

func TestTransaction_Confirm_EdgeCases_SQLStore(t *testing.T) {
	db, teardown := sqlstore.TestDB(t, databaseURL)
	defer teardown("users", "transactions")

	s := sqlstore.New(db)
	sender := &model.User{Name: "Sender", Email: "sender2@example.org", Password: "pass", AmountOfMoney: 1000}
	recipient := &model.User{Name: "Recipient", Email: "recipient2@example.org", Password: "pass", AmountOfMoney: 1000}
	assert.NoError(t, s.User().Create(sender))
	assert.NoError(t, s.User().Create(recipient))

	db.Exec("DELETE FROM transactions") // очистим транзакции
	txRepo := s.Transaction()

	// 1. Подтверждение обычной транзакции
	tx := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 100}
	assert.NoError(t, txRepo.Create(tx))
	// Подтверждаем вручную (эмулируем server.go)
	_, err := db.Exec("UPDATE transactions SET status = $1 WHERE id = $2", model.TransactionStatusConfirmed, tx.ID)
	assert.NoError(t, err)
	row := db.QueryRow("SELECT status FROM transactions WHERE id = $1", tx.ID)
	var status string
	row.Scan(&status)
	assert.Equal(t, model.TransactionStatusConfirmed, status)

	// 2. Попытка подтвердить уже отменённую транзакцию
	tx2 := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 50}
	assert.NoError(t, txRepo.Create(tx2))
	_, err = db.Exec("UPDATE transactions SET status = $1 WHERE id = $2", model.TransactionStatusCanceled, tx2.ID)
	assert.NoError(t, err)
	// Попытка подтвердить
	_, err = db.Exec("UPDATE transactions SET status = $1 WHERE id = $2 AND status = $3", model.TransactionStatusConfirmed, tx2.ID, model.TransactionStatusCreated)
	// Статус не должен измениться
	row = db.QueryRow("SELECT status FROM transactions WHERE id = $1", tx2.ID)
	row.Scan(&status)
	assert.Equal(t, model.TransactionStatusCanceled, status)

	// 3. Попытка подтвердить уже подтверждённую транзакцию
	tx3 := &model.Transaction{FromUserID: sender.ID, ToUserID: recipient.ID, AmountOfMoney: 70}
	assert.NoError(t, txRepo.Create(tx3))
	_, err = db.Exec("UPDATE transactions SET status = $1 WHERE id = $2", model.TransactionStatusConfirmed, tx3.ID)
	assert.NoError(t, err)
	// Попытка повторно подтвердить
	_, err = db.Exec("UPDATE transactions SET status = $1 WHERE id = $2 AND status = $3", model.TransactionStatusConfirmed, tx3.ID, model.TransactionStatusCreated)
	// Статус не должен измениться
	row = db.QueryRow("SELECT status FROM transactions WHERE id = $1", tx3.ID)
	row.Scan(&status)
	assert.Equal(t, model.TransactionStatusConfirmed, status)
}
