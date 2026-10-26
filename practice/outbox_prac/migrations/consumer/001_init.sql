CREATE TABLE processed_events (
    source_event_id UUID PRIMARY KEY,
    source_event_type TEXT NOT NULL,
    original_message TEXT NOT NULL,
    processed_message TEXT NOT NULL,
    source_occurred_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
