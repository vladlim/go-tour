package repository

import (
	"context"
	"strconv"
	"sync"

	"practice/grpc-service/internal/user"
)

// Memory stores users until the server stops. A mutex protects concurrent RPCs.
type Memory struct {
	mu     sync.RWMutex
	nextID uint64
	users  map[string]user.User
}

func NewMemory() *Memory {
	return &Memory{users: make(map[string]user.User)}
}

func (m *Memory) Create(ctx context.Context, name string) (user.User, error) {
	if err := ctx.Err(); err != nil {
		return user.User{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	u := user.User{ID: strconv.FormatUint(m.nextID, 10), Name: name}
	m.users[u.ID] = u
	return u, nil
}

func (m *Memory) Get(ctx context.Context, id string) (user.User, error) {
	if err := ctx.Err(); err != nil {
		return user.User{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return u, nil
}
