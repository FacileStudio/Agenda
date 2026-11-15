-- +goose Up
ALTER TABLE events ADD COLUMN external_url text NOT NULL DEFAULT '';

CREATE TABLE activities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('sablier', 'echo')),
    external_id TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ,
    url TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT activities_source_external_user UNIQUE (source, external_id, user_id)
);

CREATE INDEX activities_user_start ON activities (user_id, start_at);

-- +goose StatementBegin
CREATE FUNCTION agenda_notify_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        PERFORM pg_notify('agenda_changes', 'e:' || OLD.calendar_id);
    END IF;
    IF TG_OP <> 'DELETE' AND (TG_OP = 'INSERT' OR NEW.calendar_id IS DISTINCT FROM OLD.calendar_id) THEN
        PERFORM pg_notify('agenda_changes', 'e:' || NEW.calendar_id);
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION agenda_notify_calendar() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND (to_jsonb(NEW) - 'sync_token' - 'updated_at') = (to_jsonb(OLD) - 'sync_token' - 'updated_at') THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'DELETE' THEN
        PERFORM pg_notify('agenda_changes', 'c:' || OLD.id);
    ELSE
        PERFORM pg_notify('agenda_changes', 'c:' || NEW.id);
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION agenda_notify_calendar_member() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        PERFORM pg_notify('agenda_changes', 'c:' || OLD.calendar_id);
    END IF;
    IF TG_OP <> 'DELETE' THEN
        PERFORM pg_notify('agenda_changes', 'c:' || NEW.calendar_id);
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION agenda_notify_space_member() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        PERFORM pg_notify('agenda_changes', 'c:' || c.id) FROM calendars c WHERE c.space_id = OLD.space_id;
    END IF;
    IF TG_OP <> 'DELETE' THEN
        PERFORM pg_notify('agenda_changes', 'c:' || c.id) FROM calendars c WHERE c.space_id = NEW.space_id;
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION agenda_notify_activity() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        PERFORM pg_notify('agenda_changes', 'u:' || OLD.user_id);
    END IF;
    IF TG_OP <> 'DELETE' THEN
        PERFORM pg_notify('agenda_changes', 'u:' || NEW.user_id);
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER events_notify AFTER INSERT OR UPDATE OR DELETE ON events
    FOR EACH ROW EXECUTE FUNCTION agenda_notify_event();
CREATE TRIGGER calendars_notify AFTER INSERT OR UPDATE OR DELETE ON calendars
    FOR EACH ROW EXECUTE FUNCTION agenda_notify_calendar();
CREATE TRIGGER calendar_members_notify AFTER INSERT OR UPDATE OR DELETE ON calendar_members
    FOR EACH ROW EXECUTE FUNCTION agenda_notify_calendar_member();
CREATE TRIGGER space_members_notify AFTER INSERT OR UPDATE OR DELETE ON space_members
    FOR EACH ROW EXECUTE FUNCTION agenda_notify_space_member();
CREATE TRIGGER activities_notify AFTER INSERT OR UPDATE OR DELETE ON activities
    FOR EACH ROW EXECUTE FUNCTION agenda_notify_activity();

-- +goose Down
DROP TRIGGER IF EXISTS activities_notify ON activities;
DROP TRIGGER IF EXISTS space_members_notify ON space_members;
DROP TRIGGER IF EXISTS calendar_members_notify ON calendar_members;
DROP TRIGGER IF EXISTS calendars_notify ON calendars;
DROP TRIGGER IF EXISTS events_notify ON events;
DROP FUNCTION IF EXISTS agenda_notify_activity();
DROP FUNCTION IF EXISTS agenda_notify_space_member();
DROP FUNCTION IF EXISTS agenda_notify_calendar_member();
DROP FUNCTION IF EXISTS agenda_notify_calendar();
DROP FUNCTION IF EXISTS agenda_notify_event();
DROP TABLE IF EXISTS activities;
ALTER TABLE events DROP COLUMN IF EXISTS external_url;
