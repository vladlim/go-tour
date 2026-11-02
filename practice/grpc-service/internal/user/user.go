package user

import "errors"

var (
	ErrInvalidArgument = errors.New("id or name must not be empty")
	ErrNotFound        = errors.New("user not found")
)

type User struct {
	ID   string
	Name string
}
