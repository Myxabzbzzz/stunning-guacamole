package store

import (
	"database/sql"

	_ "github.com/lib/pq"
)

type Store struct {
	config         *Config
	db             *sql.DB
	UserRepository *UserRepository
}

func New(config *Config) *Store {
	return &Store{
		config: config,
	}
}

// Open
func (apiserver *Store) Open() error {
	db, err := sql.Open("postgres", apiserver.config.DatabaseURL)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	apiserver.db = db
	return nil
}

// Closed
func (apiserver *Store) Closed() {
	err := apiserver.db.Close()
	if err != nil {
		return
	}
}

// User ...
func (apiserver *Store) User() *UserRepository {
	if apiserver.UserRepository != nil {
		return apiserver.UserRepository
	}

	apiserver.UserRepository = &UserRepository{
		store: apiserver,
	}
	return apiserver.UserRepository
}

//store.User().Create
