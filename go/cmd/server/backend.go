package main

import (
	"context"
	"database/sql"

	"github.com/eastLaugh/web-app-go/go/internal/users"
)

type HttpBackendServer interface {
	GetUser(ctx context.Context, id int) (users.User, error)
	CreateUser(ctx context.Context, user users.User) (int, error)
}

type httpBackendServer struct {
	userRepo users.IUserRepo
	memoryKV map[string]string

	app
}

func newHttpBackendServer(userRepo users.IUserRepo) HttpBackendServer {
	return &httpBackendServer{
		userRepo: userRepo,
		memoryKV: make(map[string]string),
	}
}

func NewHttpBackendServerWithSqlite() HttpBackendServer {

	db, err := sql.Open("sqlite3", "data/user.db")
	if err != nil {
		panic(err)
	}
	// users.Migrate(db)
	return newHttpBackendServer(users.SQLiteUserRepo{Db: db})
}

func (b *httpBackendServer) GetUser(ctx context.Context, id int) (users.User, error) {
	var user users.User
	err := b.userRepo.GetByID(&user, int64(id))
	return user, err
}

func (b *httpBackendServer) CreateUser(ctx context.Context, user users.User) (int, error) {
	id, err := b.userRepo.Create(user)
	return int(id), err
}

type app struct {
}
