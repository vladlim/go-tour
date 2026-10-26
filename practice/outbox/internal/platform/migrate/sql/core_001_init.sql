CREATE TABLE posts (
    id uuid PRIMARY KEY,
    title text NOT NULL CHECK (length(title) > 0),
    body text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE TABLE idempotency_keys (
    scope text NOT NULL,
    idempotency_key text NOT NULL,
    request_hash text NOT NULL,
    response jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (scope, idempotency_key)
);
CREATE INDEX idempotency_keys_expires_at_idx ON idempotency_keys (expires_at);

CREATE TABLE outbox_events (
    id uuid PRIMARY KEY,
    event_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    topic text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    locked_until timestamptz,
    last_error text
);
CREATE INDEX outbox_pending_idx ON outbox_events (next_attempt_at, created_at) WHERE published_at IS NULL;
CREATE INDEX outbox_published_idx ON outbox_events (published_at) WHERE published_at IS NOT NULL;

-- Only coordinates deterministic crash demonstrations across container restarts.
CREATE TABLE demo_failures (
    name text PRIMARY KEY,
    triggered_at timestamptz NOT NULL DEFAULT now()
);
