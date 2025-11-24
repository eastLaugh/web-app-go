package data_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/eastLaugh/web-app-go/go/internal/users"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

func TestMigrate(t *testing.T) {
	os.Remove("user.db")

	db, err := gorm.Open(sqlite.Open("file:user.db?mode=memory"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	db.AutoMigrate(&users.User{})
}
