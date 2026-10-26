package domain

import "time"

type Item struct {
	ID               string    `json:"id"`
	SourcePostID     string    `json:"source_post_id"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
	SourceOccurredAt time.Time `json:"source_occurred_at"`
	CreatedAt        time.Time `json:"created_at"`
}
