package sqlstore_test

import (
	"billing_API/internal/app/model"
	"billing_API/internal/app/store"
	"billing_API/internal/app/store/sqlstore"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserRepository_Create(t *testing.T) {
	db, teardown := sqlstore.TestDB(t, databaseURL)
	defer teardown("users")

	s := sqlstore.New(db)
	u := model.TestUser(t)
	assert.NoError(t, s.User().Create(u))
	assert.NotNil(t, u.ID)
}

func TestUserRepository_Find(t *testing.T) {
	db, teardown := sqlstore.TestDB(t, databaseURL)
	defer teardown("users")

	s := sqlstore.New(db)
	u1 := model.TestUser(t)
	s.User().Create(u1)
	u2, err := s.User().Find(u1.ID)
	assert.NoError(t, err)
	assert.NotNil(t, u2)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	db, teardown := sqlstore.TestDB(t, databaseURL)
	defer teardown("users")

	s := sqlstore.New(db)
	u1 := model.TestUser(t)
	_, err := s.User().FindByEmail(u1.Email)
	assert.EqualError(t, err, store.ErrRecordNotFound.Error())

	s.User().Create(u1)
	u2, err := s.User().FindByEmail(u1.Email)
	assert.NoError(t, err)
	assert.NotNil(t, u2)
}

func TestUser_EdgeCases_SQLStore(t *testing.T) {
	db, teardown := sqlstore.TestDB(t, databaseURL)
	defer teardown("users")

	s := sqlstore.New(db)

	// 1. Создание с уже существующим email
	u1 := &model.User{Name: "User1", Email: "user1@example.org", Password: "password", PhoneNumber: "123", CardNumber: "111", AmountOfMoney: 100}
	assert.NoError(t, s.User().Create(u1))
	u2 := &model.User{Name: "User2", Email: "user1@example.org", Password: "password", PhoneNumber: "456", CardNumber: "222", AmountOfMoney: 200}
	err := s.User().Create(u2)
	assert.Error(t, err)

	// 2. Некорректный email
	u3 := &model.User{Name: "User3", Email: "invalid", Password: "password", PhoneNumber: "789", CardNumber: "333", AmountOfMoney: 100}
	err = s.User().Create(u3)
	assert.Error(t, err)

	// 3. Короткий пароль
	u4 := &model.User{Name: "User4", Email: "user4@example.org", Password: "123", PhoneNumber: "000", CardNumber: "444", AmountOfMoney: 100}
	err = s.User().Create(u4)
	assert.Error(t, err)

	// 4. Пустые обязательные поля
	u5 := &model.User{Name: "", Email: "user5@example.org", Password: "password", PhoneNumber: "", CardNumber: "", AmountOfMoney: 100}
	err = s.User().Create(u5)
	assert.Error(t, err)

	// 5. Отрицательный баланс
	u6 := &model.User{Name: "User6", Email: "user6@example.org", Password: "password", PhoneNumber: "123", CardNumber: "555", AmountOfMoney: -100}
	err = s.User().Create(u6)
	assert.Error(t, err)

	// 6. Невозможность входа для удалённого пользователя
	u7 := &model.User{Name: "User7", Email: "user7@example.org", Password: "password", PhoneNumber: "123", CardNumber: "666", AmountOfMoney: 100}
	assert.NoError(t, s.User().Create(u7))
	_, err = db.Exec("UPDATE users SET is_deleted = TRUE WHERE id = $1", u7.ID)
	assert.NoError(t, err)
	found, err := s.User().Find(u7.ID)
	assert.NoError(t, err)
	assert.True(t, found.IsDeleted)
}
