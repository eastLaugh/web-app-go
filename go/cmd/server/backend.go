package main

import (
	"context"

	"github.com/eastLaugh/web-app-go/go/internal/users"
)

type IBackend interface {
	GetUser(ctx context.Context, id int) (users.User, error)
	CreateUser(ctx context.Context, user users.User) (int, error)
}

type BackendServer struct {
	userRepo users.IUserRepo
	memoryKV map[string]string
}

func NewBackendServer(userRepo users.IUserRepo) IBackend {
	return BackendServer{
		userRepo: userRepo,
	}
}

func (b BackendServer) GetUser(ctx context.Context, id int) (users.User, error) {
	return b.userRepo.Get(ctx, id)
}

func (b BackendServer) CreateUser(ctx context.Context, user users.User) (int, error) {
	return b.userRepo.Create(ctx, user)
}

