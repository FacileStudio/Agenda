-- +goose Up
CREATE TABLE plans (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    space_id BIGINT REFERENCES spaces(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    warnings JSONB NOT NULL DEFAULT '[]',
    state TEXT NOT NULL DEFAULT 'proposed' CHECK (state IN ('proposed', 'applied', 'dismissed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX plans_user_created ON plans (user_id, created_at DESC);
CREATE INDEX plans_expires ON plans (expires_at);

CREATE TABLE plan_blocks (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    task_ref TEXT NOT NULL DEFAULT '',
    task_url TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    calendar_id BIGINT NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    reason TEXT NOT NULL DEFAULT '',
    event_id BIGINT REFERENCES events(id) ON DELETE SET NULL,
    CHECK (end_at > start_at)
);

CREATE INDEX plan_blocks_plan ON plan_blocks (plan_id, position);

-- +goose Down
DROP TABLE IF EXISTS plan_blocks;
DROP TABLE IF EXISTS plans;
