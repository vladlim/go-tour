package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/outbox-practice/core/internal/domain"
)

var ErrIdempotencyConflict = errors.New("idempotency key was already used with another request")

type CreatePostParams struct {
	Post           domain.Post
	EventID        string
	Topic          string
	Payload        []byte
	IdempotencyKey string
	RequestHash    string
	ExpiresAt      time.Time
}

type Repository struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) CreatePost(ctx context.Context, p CreatePostParams) (domain.Post, bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Post{}, false, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var storedHash string
	var response []byte
	err = tx.QueryRow(ctx, `
		INSERT INTO idempotency_keys (scope, idempotency_key, request_hash, response, expires_at)
		VALUES ('create-post', $1, $2, NULL, $3)
		ON CONFLICT (scope, idempotency_key) DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
		RETURNING request_hash, response`, p.IdempotencyKey, p.RequestHash, p.ExpiresAt).Scan(&storedHash, &response)
	if err != nil {
		return domain.Post{}, false, fmt.Errorf("reserve idempotency key: %w", err)
	}
	if storedHash != p.RequestHash {
		return domain.Post{}, false, ErrIdempotencyConflict
	}
	if response != nil {
		var existing domain.Post
		if err := json.Unmarshal(response, &existing); err != nil {
			return domain.Post{}, false, fmt.Errorf("decode stored response: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.Post{}, false, fmt.Errorf("commit replay: %w", err)
		}
		return existing, true, nil
	}

	if _, err = tx.Exec(ctx, `INSERT INTO posts (id, title, body, created_at) VALUES ($1,$2,$3,$4)`, p.Post.ID, p.Post.Title, p.Post.Body, p.Post.CreatedAt); err != nil {
		return domain.Post{}, false, fmt.Errorf("insert post: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events (id, event_type, aggregate_id, topic, payload) VALUES ($1,'post.created.v1',$2,$3,$4)`, p.EventID, p.Post.ID, p.Topic, p.Payload); err != nil {
		return domain.Post{}, false, fmt.Errorf("insert outbox event: %w", err)
	}
	encoded, err := json.Marshal(p.Post)
	if err != nil {
		return domain.Post{}, false, fmt.Errorf("encode response: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE idempotency_keys SET response=$3 WHERE scope='create-post' AND idempotency_key=$1 AND request_hash=$2`, p.IdempotencyKey, p.RequestHash, encoded); err != nil {
		return domain.Post{}, false, fmt.Errorf("save idempotent response: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Post{}, false, fmt.Errorf("commit create post: %w", err)
	}
	return p.Post, false, nil
}

func (r *Repository) ListPosts(ctx context.Context) ([]domain.Post, error) {
	rows, err := r.db.Query(ctx, `SELECT id,title,body,created_at FROM posts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := make([]domain.Post, 0)
	for rows.Next() {
		var p domain.Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func (r *Repository) Ping(ctx context.Context) error { return r.db.Ping(ctx) }

func (r *Repository) Cleanup(ctx context.Context, outboxBefore, idempotencyBefore time.Time, batch int) (int64, int64, error) {
	tag1, err := r.db.Exec(ctx, `DELETE FROM outbox_events WHERE id IN (SELECT id FROM outbox_events WHERE published_at IS NOT NULL AND published_at < $1 ORDER BY published_at LIMIT $2)`, outboxBefore, batch)
	if err != nil {
		return 0, 0, err
	}
	tag2, err := r.db.Exec(ctx, `DELETE FROM idempotency_keys WHERE expires_at < $1`, idempotencyBefore)
	return tag1.RowsAffected(), tag2.RowsAffected(), err
}
