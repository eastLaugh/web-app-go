package users

import (
	"database/sql"
)

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"` //只使用邮箱+验证码登录
	Name  string `json:"name"`
}

type IUserRepo interface {
	GetByID(*User, int64) error
	Create(user User) (int64, error)
}

var _ IUserRepo = SQLiteUserRepo{}

type SQLiteUserRepo struct {
	Db *sql.DB
}

func (repo SQLiteUserRepo) Create(user User) (int64, error) {
	result, err := repo.Db.
		Exec("INSERT INTO users (email, name) VALUES (?, ?)", user.Email, user.Name)

	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (repo SQLiteUserRepo) GetByID(user *User, id int64) error {
	err := repo.Db.
		QueryRow("SELECT id, email, name FROM users WHERE id = ?", id).
		Scan(&user.ID, &user.Email, &user.Name)
	return err

}
