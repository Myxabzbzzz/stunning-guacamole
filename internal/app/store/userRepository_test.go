package store_test

import (
	"dodobackend/cmd/model"
	_ "dodobackend/internal/app/apiserver"
	"dodobackend/internal/app/store"
	"testing"
)

func TestUserRepository_Create(t *testing.T) {
	s, teardown := store.SetupTestStore(t, databaseURL)
	defer teardown("users")
	u, err := s.User().Create(&model.User{
		Name:       "shukur",
		Email:      "shukur@gmail.com",
		CardNumber: "1234123412341234",
		Balance:    300,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID == 0 {
		t.Error("expected user ID to be set")
	}
}

func TestUserRepository_FindByEmail(t *testing.T) {
	s, teardown := store.SetupTestStore(t, databaseURL)
	defer teardown("users")
	u, err := s.User().FindByEmail("shukur@gmail.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID == 0 {
	}
}
