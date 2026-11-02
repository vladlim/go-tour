package grpc

import (
	"context"
	"errors"

	userv1 "practice/grpc-service/api/user/v1"
	"practice/grpc-service/internal/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service interface {
	Create(context.Context, string) (user.User, error)
	Get(context.Context, string) (user.User, error)
}

type Handler struct {
	userv1.UnimplementedUserServiceServer
	service Service
}

func New(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.User, error) {
	u, err := h.service.Create(ctx, req.GetName())
	if err != nil {
		return nil, rpcError(err)
	}
	return &userv1.User{Id: u.ID, Name: u.Name}, nil
}

func (h *Handler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.User, error) {
	u, err := h.service.Get(ctx, req.GetId())
	if err != nil {
		return nil, rpcError(err)
	}
	return &userv1.User{Id: u.ID, Name: u.Name}, nil
}

func rpcError(err error) error {
	switch {
	case errors.Is(err, user.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, user.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
