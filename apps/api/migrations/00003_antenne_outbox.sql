-- +goose Up
CREATE TABLE antenne_outbox (
    id BIGSERIAL PRIMARY KEY,
    channel TEXT NOT NULL,
    payload JSONB NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ
);

CREATE INDEX antenne_outbox_unsent ON antenne_outbox (id) WHERE sent_at IS NULL;
CREATE INDEX antenne_outbox_sent ON antenne_outbox (sent_at) WHERE sent_at IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS antenne_outbox;
