package event

import "time"

const PostCreatedV1 = "post.created.v1"

type Envelope[T any] struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"`
	AggregateID      string    `json:"aggregate_id"`
	AggregateVersion int       `json:"aggregate_version"`
	OccurredAt       time.Time `json:"occurred_at"`
	Data             T         `json:"data"`
}

type PostCreated struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
