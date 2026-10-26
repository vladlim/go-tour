CREATE TABLE feed_items (
    id uuid PRIMARY KEY,
    source_post_id uuid NOT NULL UNIQUE,
    title text NOT NULL,
    body text NOT NULL,
    source_occurred_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE inbox_events (
    event_id uuid PRIMARY KEY,
    event_type text NOT NULL,
    topic text NOT NULL,
    partition integer NOT NULL,
    message_offset bigint NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX inbox_processed_at_idx ON inbox_events (processed_at);
