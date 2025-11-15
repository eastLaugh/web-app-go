package data_test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestResetDB(t *testing.T) {
	os.Remove("user.db")

	db, err := sql.Open("sqlite3", "user.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT UNIQUE, name TEXT)
		`,
	)
	db.Exec(
		`INSERT INTO users (email, name) VALUES (?, ?)`,
		"test@test.com",
		"test",
	)

}
