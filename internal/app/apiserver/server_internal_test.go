package apiserver

import (
	"billing_API/internal/app/model"
	"billing_API/internal/app/store/teststore"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
)

func TestServer_AuthenticateUser(t *testing.T) {
	store := teststore.New()
	u := model.TestUser(t)
	err := store.User().Create(u)
	if err != nil {
		return
	}

	testCases := []struct {
		name         string
		cookieValue  map[interface{}]interface{}
		expectedCode int
	}{
		{
			name: "authenticated",
			cookieValue: map[interface{}]interface{}{
				"user_id": u.ID,
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "not authenticated",
			cookieValue:  nil,
			expectedCode: http.StatusUnauthorized,
		},
	}

	secretKey := []byte("secret")
	s := newServer(store, sessions.NewCookieStore(secretKey))
	sc := securecookie.New(secretKey, nil)
	mw := s.authenticateUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			cookieStr, _ := sc.Encode(sessionName, tc.cookieValue)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", sessionName, cookieStr))
			mw.ServeHTTP(rec, req)
			assert.Equal(t, tc.expectedCode, rec.Code)
		})
	}
}

func TestServer_HandleUsersCreate(t *testing.T) {
	s := newServer(teststore.New(), sessions.NewCookieStore([]byte("secret")))
	testCases := []struct {
		name         string
		payload      interface{}
		expectedCode int
	}{
		{
			name: "valid",
			payload: map[string]interface{}{
				"email":    "user@example.org",
				"password": "secret",
			},
			expectedCode: http.StatusCreated,
		},
		{
			name:         "invalid payload",
			payload:      "invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid params",
			payload: map[string]interface{}{
				"email":    "invalid",
				"password": "short",
			},
			expectedCode: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bytes.Buffer{}
			err := json.NewEncoder(b).Encode(tc.payload)
			if err != nil {
				return
			}
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/users", b)
			s.ServeHTTP(rec, req)
			assert.Equal(t, tc.expectedCode, rec.Code)
		})
	}
}

func TestServer_HandleSessionsCreate(t *testing.T) {
	store := teststore.New()
	u := model.TestUser(t)
	err := store.User().Create(u)
	if err != nil {
		return
	}
	s := newServer(store, sessions.NewCookieStore([]byte("secret")))
	testCases := []struct {
		name         string
		payload      interface{}
		expectedCode int
	}{
		{
			name: "valid",
			payload: map[string]interface{}{
				"email":    u.Email,
				"password": u.Password,
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "invalid payload",
			payload:      "invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid email",
			payload: map[string]interface{}{
				"email":    "invalid",
				"password": u.Password,
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "invalid password",
			payload: map[string]interface{}{
				"email":    u.Email,
				"password": "invalid",
			},
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bytes.Buffer{}
			err := json.NewEncoder(b).Encode(tc.payload)
			if err != nil {
				return
			}
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/sessions", b)
			s.ServeHTTP(rec, req)
			assert.Equal(t, tc.expectedCode, rec.Code)
		})
	}
}

func TestAuth_EdgeCases(t *testing.T) {
	s := newServer(teststore.New(), sessions.NewCookieStore([]byte("secret")))
	store := teststore.New()
	repo := store.User().(*teststore.UserRepository)
	user := model.TestUser(t)
	user.Email = "edgecase@example.org"
	user.Password = "password123"
	repo.Create(user)

	// 1. Login with incorrect email
	b := &bytes.Buffer{}
	json.NewEncoder(b).Encode(map[string]interface{}{"email": "wrong@example.org", "password": "password123"})
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/sessions", b)
	s.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 2. Login with incorrect password
	b.Reset()
	json.NewEncoder(b).Encode(map[string]interface{}{"email": user.Email, "password": "wrongpass"})
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/sessions", b)
	s.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 3. Login for deleted user
	user.IsDeleted = true
	repo.Users[user.ID] = user
	b.Reset()
	json.NewEncoder(b).Encode(map[string]interface{}{"email": user.Email, "password": "password123"})
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/sessions", b)
	s.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	user.IsDeleted = false // revert

	// 4. Access private endpoint without session
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/private/whoami", nil)
	s.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// 5. Access private endpoint with valid session
	secretKey := []byte("secret")
	srv := newServer(store, sessions.NewCookieStore(secretKey))
	sc := securecookie.New(secretKey, nil)
	cookieValue := map[interface{}]interface{}{"user_id": user.ID}
	cookieStr, _ := sc.Encode(sessionName, cookieValue)
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/private/whoami", nil)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", sessionName, cookieStr))
	srv.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// 6. Access private endpoint after deleting user
	user.IsDeleted = true
	repo.Users[user.ID] = user
	rec = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/private/whoami", nil)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", sessionName, cookieStr))
	srv.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
