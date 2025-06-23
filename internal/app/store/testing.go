package store

import (
	"fmt"
	"strings"
	"testing"
)

// SetupTestStore

func SetupTestStore(t *testing.T, databaseURL string) (*Store, func(...string)) {
	t.Helper()
	config := NewConfig()
	config.DatabaseURL = databaseURL
	s := New(config)
	if err := s.Open(); err != nil {
		t.Fatal(err)
	}
	return s, func(tables ...string) {
		if len(tables) > 0 {
			query := fmt.Sprintf("TRUNCATE %s CASCADE", strings.Join(tables, ","))
			if _, err := s.db.Exec(query); err != nil {
				t.Fatal(err)
			}
		}
		s.Closed()
	}
}
