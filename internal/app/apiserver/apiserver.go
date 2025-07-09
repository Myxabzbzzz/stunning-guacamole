package apiserver

import (
	"billing_API/internal/app/store/sqlstore"
	"database/sql"
	"net/http"

	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Start ...
func Start(config *Config) error {
	db, err := newDB(config.DatabaseURL)
	if err != nil {
		return err
	}

	defer db.Close()
	store := sqlstore.New(db)
	sessionStore := sessions.NewCookieStore([]byte(config.SessionKey))
	srv := newServer(store, sessionStore)

	return http.ListenAndServe(config.BindAddr, srv)
}

// StartWithSwagger
func StartWithSwagger(config *Config) error {
	db, err := newDB(config.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	store := sqlstore.New(db)
	sessionStore := sessions.NewCookieStore([]byte(config.SessionKey))
	srv := newServer(store, sessionStore)

	srv.router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Добавляем обработчик для /swagger.yaml
	srv.router.PathPrefix("/swagger.yaml").Handler(http.StripPrefix("/swagger.yaml", http.FileServer(http.Dir("docs"))))

	return http.ListenAndServe(config.BindAddr, srv)
}

func newDB(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
