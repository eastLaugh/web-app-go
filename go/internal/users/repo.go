package users

import (
	"context"
	"database/sql"
)

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"` //只使用邮箱+验证码登录
	Name  string `json:"name"`
}

type IUserRepo interface {
	Get(ctx context.Context, id int) (User, error)
	Create(ctx context.Context, user User) (int, error)
}

var _ IUserRepo = SQLiteUserRepo{}

type SQLiteUserRepo struct {
	Db *sql.DB
}

func (repo SQLiteUserRepo) Create(ctx context.Context, user User) (int, error) {
	result, err := repo.Db.
		ExecContext(ctx, "INSERT INTO users (email, name) VALUES (?, ?)", user.Email, user.Name)

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (repo SQLiteUserRepo) Get(ctx context.Context, id int) (User, error) {
	user := User{}
	err := repo.Db.
		QueryRowContext(ctx, "SELECT id, email, name FROM users WHERE id = ?", id).
		Scan(&user.ID, &user.Email, &user.Name)
	if err != nil {
		return User{}, err
	}
	return user, nil

}

