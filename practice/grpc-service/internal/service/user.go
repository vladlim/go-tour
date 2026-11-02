package service

import (
	"context"
	"strings"

	"practice/grpc-service/internal/user"
)

// Repository is defined by the business layer that consumes it.
type Repository interface {
	Create(context.Context, string) (user.User, error)
	Get(context.Context, string) (user.User, error)
}

type UserService struct{ repo Repository }

func New(repo Repository) *UserService { return &UserService{repo: repo} }

func (s *UserService) Create(ctx context.Context, name string) (user.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return user.User{}, user.ErrInvalidArgument
	}
	return s.repo.Create(ctx, name)
}

func (s *UserService) Get(ctx context.Context, id string) (user.User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return user.User{}, user.ErrInvalidArgument
	}
	return s.repo.Get(ctx, id)
}
