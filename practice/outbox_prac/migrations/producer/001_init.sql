CREATE TABLE events (
    id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    message TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE outbox (
    id UUID PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES events(id),
    topic TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error TEXT
);

CREATE INDEX outbox_pending_idx ON outbox (next_attempt_at, created_at) WHERE published_at IS NULL;
