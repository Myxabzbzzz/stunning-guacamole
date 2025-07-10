package apiserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"

	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/gorilla/handlers"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"billing_API/internal/app/model"
	"billing_API/internal/app/store"
	"billing_API/internal/app/store/sqlstore"
	"sync"
)

const (
	sessionName        = "myxabzbzzz"
	ctxKeyUser  ctxKey = iota
	ctxKeyRequestID
)

var (
	errIncorrectEmailOrPassword = errors.New("incorrect email or password")
	errNotAuthenticated         = errors.New("not authenticated")
)

type ctxKey int8

type server struct {
	router       *mux.Router
	logger       *logrus.Logger
	store        store.Store
	sessionStore sessions.Store
}

func newServer(store store.Store, sessionStore sessions.Store) *server {
	s := &server{
		router:       mux.NewRouter(),
		logger:       logrus.New(),
		store:        store,
		sessionStore: sessionStore,
	}

	s.configureRouter()

	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) configureRouter() {
	s.router.Use(s.setRequestID)
	s.router.Use(s.logRequest)
	s.router.Use(handlers.CORS(handlers.AllowedOrigins([]string{"*"})))
	s.router.Use(s.rateLimitMiddleware(10, time.Second))
	s.router.HandleFunc("/users", s.handleUsersCreate()).Methods("POST")
	s.router.HandleFunc("/sessions", s.handleSessionsCreate()).Methods("POST")

	s.router.HandleFunc("/transactions", s.handleTransactionsCreate()).Methods("POST")
	s.router.HandleFunc("/transactions", s.handleTransactionsList()).Methods("GET")
	s.router.HandleFunc("/transactions/{id}/confirm", s.handleTransactionsConfirm()).Methods("POST")
	s.router.HandleFunc("/transactions/{id}/cancel", s.handleTransactionCancel()).Methods("POST")

	s.router.HandleFunc("/status", s.handleStatus()).Methods("GET")

	s.router.HandleFunc("/users/{id}", s.handleGetUserByID()).Methods("GET")
	s.router.HandleFunc("/users/{id}/balance", s.handleGetUserBalance()).Methods("GET")
	s.router.HandleFunc("/users/{id}/transactions", s.handleGetUserTransactions()).Methods("GET")
	s.router.HandleFunc("/users", s.handleListUsers()).Methods("GET")
	s.router.HandleFunc("/users/{id}/delete", s.handleSoftDeleteUser()).Methods("POST")
	s.router.HandleFunc("/users/{id}/restore", s.handleRestoreUser()).Methods("POST")
	s.router.HandleFunc("/transactions/{id}", s.handleGetTransactionByID()).Methods("GET")

	private := s.router.PathPrefix("/private").Subrouter()
	private.Use(s.authenticateUser)
	private.HandleFunc("/whoami", s.handleWhoami())
}

func (s *server) setRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyRequestID, id)))
	})
}

func (s *server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := s.logger.WithFields(logrus.Fields{
			"remote_addr": r.RemoteAddr,
			"request_id":  r.Context().Value(ctxKeyRequestID),
		})
		logger.Infof("started %s %s", r.Method, r.RequestURI)

		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		var level logrus.Level
		switch {
		case rw.code >= 500:
			level = logrus.ErrorLevel
		case rw.code >= 400:
			level = logrus.WarnLevel
		default:
			level = logrus.InfoLevel
		}
		logger.Logf(
			level,
			"completed with %d %s in %v",
			rw.code,
			http.StatusText(rw.code),
			time.Now().Sub(start),
		)
	})
}

func (s *server) authenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessionStore.Get(r, sessionName)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		id, ok := session.Values["user_id"]
		if !ok {
			s.error(w, r, http.StatusUnauthorized, errNotAuthenticated)
			return
		}

		u, err := s.store.User().Find(id.(int))
		if err != nil {
			s.error(w, r, http.StatusUnauthorized, errNotAuthenticated)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUser, u)))
	})
}

func (s *server) handleUsersCreate() http.HandlerFunc {
	type request struct {
		Name          string `json:"name"`
		PhoneNumber   string `json:"phone_number"`
		CardNumber    string `json:"card_number"`
		AmountOfMoney int64  `json:"amount_of_money"`
		Email         string `json:"email"`
		Password      string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req := &request{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}

		u := &model.User{
			Name:          req.Name,
			PhoneNumber:   req.PhoneNumber,
			CardNumber:    req.CardNumber,
			AmountOfMoney: req.AmountOfMoney,
			Email:         req.Email,
			Password:      req.Password,
		}
		if err := s.store.User().Create(u); err != nil {
			s.error(w, r, http.StatusUnprocessableEntity, err)
			return
		}

		createdUser, err := s.store.User().Find(u.ID)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		createdUser.Sanitize()
		s.respond(w, r, http.StatusCreated, createdUser)
	}
}

func (s *server) handleSessionsCreate() http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req := &request{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}

		u, err := s.store.User().FindByEmail(req.Email)
		if err != nil || !u.ComparePassword(req.Password) {
			s.error(w, r, http.StatusUnauthorized, errIncorrectEmailOrPassword)
			return
		}

		if err := u.CheckNotDeleted(); err != nil {
			s.error(w, r, http.StatusForbidden, err) // 403 — доступ запрещён
			return
		}

		session, err := s.sessionStore.Get(r, sessionName)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		session.Values["user_id"] = u.ID
		if err := s.sessionStore.Save(r, w, session); err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.respond(w, r, http.StatusOK, nil)
	}
}

func (s *server) handleWhoami() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, http.StatusOK, r.Context().Value(ctxKeyUser).(*model.User))
	}
}

func (s *server) handleTransactionsCreate() http.HandlerFunc {
	type request struct {
		FromUserID    int   `json:"from_user_id"`
		ToUserID      int   `json:"to_user_id"`
		AmountOfMoney int64 `json:"amount_of_money"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Authorization (example: only authorized user can create)
		sess, err := s.sessionStore.Get(r, sessionName)
		if err != nil || sess.Values["user_id"] == nil {
			s.error(w, r, http.StatusUnauthorized, errors.New("not authenticated"))
			return
		}
		req := &request{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid request format"))
			return
		}
		if req.AmountOfMoney <= 0 {
			s.error(w, r, http.StatusBadRequest, errors.New("amount_of_money must be positive"))
			return
		}
		if req.FromUserID == req.ToUserID {
			s.error(w, r, http.StatusBadRequest, errors.New("sender and recipient cannot be the same"))
			return
		}
		fromUser, err := s.store.User().Find(req.FromUserID)
		if err != nil || fromUser.IsDeleted {
			s.error(w, r, http.StatusBadRequest, errors.New("sender not found or deleted"))
			return
		}
		toUser, err := s.store.User().Find(req.ToUserID)
		if err != nil || toUser.IsDeleted {
			s.error(w, r, http.StatusBadRequest, errors.New("recipient not found or deleted"))
			return
		}
		var txID int
		var txTime time.Time
		db := s.store.(*sqlstore.Store).DB()
		err = db.QueryRow(
			`INSERT INTO transactions (from_user_id, to_user_id, amount_of_money, status)
			 VALUES ($1, $2, $3, $4) RETURNING id, transaction_time`,
			req.FromUserID, req.ToUserID, req.AmountOfMoney, model.TransactionStatusCreated,
		).Scan(&txID, &txTime)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		s.respond(w, r, http.StatusCreated, map[string]interface{}{
			"id":               txID,
			"from_user_id":     req.FromUserID,
			"to_user_id":       req.ToUserID,
			"amount_of_money":  req.AmountOfMoney,
			"transaction_time": txTime,
			"status":           model.TransactionStatusCreated,
		})
	}
}

func (s *server) handleTransactionsConfirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Authorization
		sess, err := s.sessionStore.Get(r, sessionName)
		if err != nil || sess.Values["user_id"] == nil {
			s.error(w, r, http.StatusUnauthorized, errors.New("not authenticated"))
			return
		}

		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid transaction id"))
			return
		}

		db := s.store.(*sqlstore.Store).DB()
		tx, err := db.Begin()
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		defer func(tx *sql.Tx) {
			err := tx.Rollback()
			if err != nil {
				s.error(w, r, http.StatusInternalServerError, err)
			}
		}(tx)

		var fromID, toID int
		var amount int64
		var status string

		err = tx.QueryRow(
			`SELECT from_user_id, to_user_id, amount_of_money, status FROM transactions
			 WHERE id = $1 FOR UPDATE`, id,
		).Scan(&fromID, &toID, &amount, &status)
		if err != nil {
			s.error(w, r, http.StatusNotFound, errors.New("transaction not found"))
			return
		}
		if status != model.TransactionStatusPending && status != model.TransactionStatusCreated {
			s.error(w, r, http.StatusBadRequest, errors.New("transaction already processed or not in pending/created status"))
			return
		}

		var senderBalance int64
		var senderDeleted, recipientDeleted bool

		err = tx.QueryRow(
			"SELECT amount_of_money, is_deleted FROM users WHERE id = $1 FOR UPDATE",
			fromID,
		).Scan(&senderBalance, &senderDeleted)
		if err != nil || senderDeleted {
			s.error(w, r, http.StatusForbidden, errors.New("invalid or deleted sender"))
			return
		}

		err = tx.QueryRow(
			"SELECT is_deleted FROM users WHERE id = $1 FOR UPDATE",
			toID,
		).Scan(&recipientDeleted)
		if err != nil || recipientDeleted {
			s.error(w, r, http.StatusForbidden, errors.New("invalid or deleted recipient"))
			return
		}

		if senderBalance < amount {
			s.error(w, r, http.StatusUnprocessableEntity, errors.New("insufficient funds"))
			return
		}

		_, err = tx.Exec("UPDATE users SET amount_of_money = amount_of_money - $1 WHERE id = $2", amount, fromID)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		_, err = tx.Exec("UPDATE users SET amount_of_money = amount_of_money + $1 WHERE id = $2", amount, toID)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		_, err = tx.Exec("UPDATE transactions SET status = $1 WHERE id = $2", model.TransactionStatusConfirmed, id)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		err = tx.Commit()
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.respond(w, r, http.StatusOK, map[string]string{"status": "confirmed"})
	}
}
func (s *server) handleTransactionsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transactions, err := s.store.Transaction().List()
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		s.respond(w, r, http.StatusOK, transactions)
	}
}

func (s *server) handleTransactionCancel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Authorization
		sess, err := s.sessionStore.Get(r, sessionName)
		if err != nil || sess.Values["user_id"] == nil {
			s.error(w, r, http.StatusUnauthorized, errors.New("not authenticated"))
			return
		}

		vars := mux.Vars(r)
		idStr, ok := vars["id"]
		if !ok {
			s.error(w, r, http.StatusBadRequest, errors.New("missing transaction id"))
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid transaction id"))
			return
		}

		result, err := s.store.(*sqlstore.Store).DB().Exec(`
			UPDATE transactions 
			SET status = $1 
			WHERE id = $2 AND status = $3`, model.TransactionStatusCanceled, id, model.TransactionStatusCreated)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		if rowsAffected == 0 {
			s.error(w, r, http.StatusBadRequest, errors.New("transaction cannot be canceled (not found or already processed)"))
			return
		}

		s.respond(w, r, http.StatusOK, map[string]string{"status": "canceled"})
	}
}

func (s *server) handleStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func (s *server) handleGetUserByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid user id"))
			return
		}
		u, err := s.store.User().Find(id)
		if err != nil {
			s.error(w, r, http.StatusNotFound, err)
			return
		}
		s.respond(w, r, http.StatusOK, u)
	}
}

func (s *server) handleGetUserBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid user id"))
			return
		}
		u, err := s.store.User().Find(id)
		if err != nil {
			s.error(w, r, http.StatusNotFound, err)
			return
		}
		s.respond(w, r, http.StatusOK, map[string]interface{}{"balance": u.AmountOfMoney})
	}
}

func (s *server) handleGetUserTransactions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid user id"))
			return
		}
		all, err := s.store.Transaction().List()
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		var userTxs []*model.Transaction
		for _, tx := range all {
			if tx.FromUserID == id || tx.ToUserID == id {
				userTxs = append(userTxs, tx)
			}
		}
		s.respond(w, r, http.StatusOK, userTxs)
	}
}

func (s *server) handleListUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := s.store.(*sqlstore.Store).DB()
		rows, err := db.Query("SELECT id, name, email, phone_number, card_number, amount_of_money, is_deleted FROM users")
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		defer func(rows *sql.Rows) {
			err := rows.Close()
			if err != nil {
				s.error(w, r, http.StatusInternalServerError, err)
			}
		}(rows)
		var users []map[string]interface{}
		for rows.Next() {
			var id int
			var name, email, phone, card string
			var amount int64
			var isDeleted bool
			if err := rows.Scan(&id, &name, &email, &phone, &card, &amount, &isDeleted); err != nil {
				s.error(w, r, http.StatusInternalServerError, err)
				return
			}
			users = append(users, map[string]interface{}{
				"id": id, "name": name, "email": email, "phone_number": phone, "card_number": card, "amount_of_money": amount, "is_deleted": isDeleted,
			})
		}
		s.respond(w, r, http.StatusOK, users)
	}
}

func (s *server) handleSoftDeleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Authorization
		sess, err := s.sessionStore.Get(r, sessionName)
		if err != nil || sess.Values["user_id"] == nil {
			s.error(w, r, http.StatusUnauthorized, errors.New("not authenticated"))
			return
		}

		idStr := mux.Vars(r)["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid user id"))
			return
		}
		db := s.store.(*sqlstore.Store).DB()
		_, err = db.Exec("UPDATE users SET is_deleted = TRUE WHERE id = $1", id)
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		s.respond(w, r, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func (s *server) handleRestoreUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.sessionStore.Get(r, sessionName)
		if err != nil || sess.Values["user_id"] == nil {
			s.error(w, r, http.StatusUnauthorized, errors.New("not authenticated"))
			return
		}
		idStr := mux.Vars(r)["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid user id"))
			return
		}
		err = s.store.User().Restore(id)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}
		s.respond(w, r, http.StatusOK, map[string]string{"status": "restored"})
	}
}

func (s *server) handleGetTransactionByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			s.error(w, r, http.StatusBadRequest, errors.New("invalid transaction id"))
			return
		}
		all, err := s.store.Transaction().List()
		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		for _, tx := range all {
			if tx.ID == id {
				s.respond(w, r, http.StatusOK, tx)
				return
			}
		}
		s.error(w, r, http.StatusNotFound, errors.New("transaction not found"))
	}
}

func (s *server) error(w http.ResponseWriter, r *http.Request, code int, err error) {
	if code >= 500 {
		s.logger.Errorf("%s %s: %v", r.Method, r.URL.Path, err)
	} else {
		s.logger.Warnf("%s %s: %v", r.Method, r.URL.Path, err)
	}
	s.respond(w, r, code, map[string]string{"error": err.Error()})
}

func (s *server) respond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.WriteHeader(code)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			s.logger.Warnf("%s %s: failed to encode response: %v", r.Method, r.URL.Path, err)
			return
		}
	}
}

// rateLimitMiddleware limits the number of requests from a single IP
func (s *server) rateLimitMiddleware(maxReq int, window time.Duration) func(http.Handler) http.Handler {
	type client struct {
		mu    sync.Mutex
		count int
		start time.Time
	}
	clients := make(map[string]*client)
	var mu sync.Mutex
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			mu.Lock()
			c, ok := clients[ip]
			if !ok {
				c = &client{start: time.Now()}
				clients[ip] = c
			}
			mu.Unlock()
			c.mu.Lock()
			if time.Since(c.start) > window {
				c.start = time.Now()
				c.count = 0
			}
			c.count++
			if c.count > maxReq {
				c.mu.Unlock()
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			c.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}
