package model

import "time"

const EventType = "demo.event.created"

type Event struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurred_at"`
}
