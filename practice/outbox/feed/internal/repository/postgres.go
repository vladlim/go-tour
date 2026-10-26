package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/outbox-practice/feed/internal/domain"
	"github.com/example/outbox-practice/internal/event"
)

type Metadata struct {
	Topic     string
	Partition int
	Offset    int64
}
type Repository struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) ApplyPostCreated(ctx context.Context, e event.Envelope[event.PostCreated], m Metadata) (bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `INSERT INTO inbox_events(event_id,event_type,topic,partition,message_offset) VALUES($1,$2,$3,$4,$5) ON CONFLICT(event_id) DO NOTHING`, e.ID, e.Type, m.Topic, m.Partition, m.Offset)
	if err != nil {
		return false, fmt.Errorf("insert inbox: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return true, nil
	}
	_, err = tx.Exec(ctx, `INSERT INTO feed_items(id,source_post_id,title,body,source_occurred_at) VALUES(gen_random_uuid(),$1,$2,$3,$4) ON CONFLICT(source_post_id) DO NOTHING`, e.AggregateID, e.Data.Title, e.Data.Body, e.OccurredAt)
	if err != nil {
		return false, fmt.Errorf("insert feed item: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit feed transaction: %w", err)
	}
	return false, nil
}

func (r *Repository) List(ctx context.Context) ([]domain.Item, error) {
	rows, err := r.db.Query(ctx, `SELECT id,source_post_id,title,body,source_occurred_at,created_at FROM feed_items ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Item, 0)
	for rows.Next() {
		var i domain.Item
		if err := rows.Scan(&i.ID, &i.SourcePostID, &i.Title, &i.Body, &i.SourceOccurredAt, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
func (r *Repository) Ping(ctx context.Context) error { return r.db.Ping(ctx) }
func (r *Repository) Cleanup(ctx context.Context, before time.Time, batch int) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM inbox_events WHERE event_id IN (SELECT event_id FROM inbox_events WHERE processed_at<$1 ORDER BY processed_at LIMIT $2)`, before, batch)
	return tag.RowsAffected(), err
}
