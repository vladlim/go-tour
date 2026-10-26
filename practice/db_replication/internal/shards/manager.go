package shards

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Shard struct {
	ID      int
	Primary *pgxpool.Pool
	Replica *pgxpool.Pool
}

type Manager struct{ shards []Shard }

func New(ctx context.Context) (*Manager, error) {
	defaults := [][2]string{
		{"postgres://app:app@localhost:5432/app?sslmode=disable", "postgres://app:app@localhost:5433/app?sslmode=disable"},
		{"postgres://app:app@localhost:5434/app?sslmode=disable", "postgres://app:app@localhost:5435/app?sslmode=disable"},
	}
	var result []Shard
	for i, pair := range defaults {
		primary, err := connect(ctx, env(fmt.Sprintf("SHARD%d_PRIMARY_DSN", i), pair[0]))
		if err != nil {
			closeShards(result)
			return nil, fmt.Errorf("connect shard %d primary: %w", i, err)
		}
		replica, err := connect(ctx, env(fmt.Sprintf("SHARD%d_REPLICA_DSN", i), pair[1]))
		if err != nil {
			primary.Close()
			closeShards(result)
			return nil, fmt.Errorf("connect shard %d replica: %w", i, err)
		}
		result = append(result, Shard{ID: i, Primary: primary, Replica: replica})
	}
	return &Manager{shards: result}, nil
}

func connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func (m *Manager) ByUserID(id int64) Shard { return m.shards[int(id%int64(len(m.shards)))] }
func (m *Manager) All() []Shard            { return m.shards }
func (m *Manager) Close()                  { closeShards(m.shards) }
func closeShards(ss []Shard) {
	for _, s := range ss {
		s.Primary.Close()
		s.Replica.Close()
	}
}
