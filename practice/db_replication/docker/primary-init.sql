CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD 'replica';

CREATE TABLE users (
    id bigint PRIMARY KEY,
    name text NOT NULL,
    email text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE orders (
    id bigint GENERATED ALWAYS AS IDENTITY,
    user_id bigint NOT NULL REFERENCES users(id),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, id)
);

CREATE INDEX orders_user_created_idx ON orders (user_id, created_at DESC);

