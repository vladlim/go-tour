package store

import (
	"context"
	"fmt"
	"time"

	"dbreplication/internal/shards"
)

type User struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
	Shard     int
	Source    string
}
type Status struct {
	Shard                           int
	PrimaryRecovery, ReplicaRecovery bool
	ReplicationLag                  *time.Duration
}
type Store struct{ manager *shards.Manager }

func New(m *shards.Manager) *Store { return &Store{manager: m} }

func (s *Store) CreateUser(ctx context.Context, id int64, name, email string) (User, error) {
	shard := s.manager.ByUserID(id)
	u := User{ID: id, Name: name, Email: email, Shard: shard.ID, Source: "primary"}
	err := shard.Primary.QueryRow(ctx, `INSERT INTO users(id,name,email) VALUES($1,$2,$3) RETURNING created_at`, id, name, email).Scan(&u.CreatedAt)
	return u, err
}

func (s *Store) GetUser(ctx context.Context, id int64, primary bool) (User, error) {
	shard := s.manager.ByUserID(id)
	pool := shard.Replica
	source := "replica"
	if primary {
		pool = shard.Primary
		source = "primary"
	}
	u := User{Shard: shard.ID, Source: source}
	err := pool.QueryRow(ctx, `SELECT id,name,email,created_at FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
	return u, err
}

func (s *Store) AllUsers(ctx context.Context) ([]User, error) {
	var out []User
	for _, shard := range s.manager.All() {
		rows, err := shard.Replica.Query(ctx, `SELECT id,name,email,created_at FROM users ORDER BY id`)
		if err != nil {
			return nil, fmt.Errorf("shard %d: %w", shard.ID, err)
		}
		for rows.Next() {
			var u User
			u.Shard, u.Source = shard.ID, "replica"
			if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, u)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return out, nil
}

func (s *Store) Status(ctx context.Context) ([]Status, error) {
	var out []Status
	for _, shard := range s.manager.All() {
		st := Status{Shard: shard.ID}
		if err := shard.Primary.QueryRow(ctx, `SELECT pg_is_in_recovery()`).Scan(&st.PrimaryRecovery); err != nil {
			return nil, err
		}
		var seconds *float64
		if err := shard.Replica.QueryRow(ctx, `SELECT pg_is_in_recovery(), EXTRACT(EPOCH FROM now()-pg_last_xact_replay_timestamp())`).Scan(&st.ReplicaRecovery, &seconds); err != nil {
			return nil, err
		}
		if seconds != nil {
			d := time.Duration(*seconds * float64(time.Second))
			st.ReplicationLag = &d
		}
		out = append(out, st)
	}
	return out, nil
}
