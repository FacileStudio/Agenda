-- +goose Up
-- baseline: the production schema as GORM AutoMigrate and schemas.MigrateWithIssuer built it, captured 2026-11-10 with pg_dump --schema-only --no-owner --no-privileges

CREATE TABLE app_settings (
    id bigint NOT NULL,
    encryption_key text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE app_settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE app_settings_id_seq OWNED BY app_settings.id;

CREATE TABLE calendar_members (
    id bigint NOT NULL,
    calendar_id bigint,
    user_id bigint,
    role text,
    created_at timestamp with time zone
);

CREATE SEQUENCE calendar_members_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE calendar_members_id_seq OWNED BY calendar_members.id;

CREATE TABLE calendars (
    id bigint NOT NULL,
    owner_id bigint,
    slug text,
    name text,
    color text,
    description text,
    is_personal boolean DEFAULT false,
    sync_token text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    echo_url text,
    space_id bigint,
    cal_dav_path text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE calendars_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE calendars_id_seq OWNED BY calendars.id;

CREATE TABLE directory_deletion_state (
    id smallint NOT NULL,
    state bytea NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT directory_deletion_state_id_check CHECK ((id = 1))
);

CREATE TABLE event_attendees (
    id bigint NOT NULL,
    event_id bigint,
    user_id bigint,
    email text,
    response text DEFAULT 'needs-action'::text,
    created_at timestamp with time zone
);

CREATE SEQUENCE event_attendees_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE event_attendees_id_seq OWNED BY event_attendees.id;

CREATE TABLE events (
    id bigint NOT NULL,
    calendar_id bigint,
    uid text,
    etag text,
    sequence bigint DEFAULT 0,
    title text,
    description text,
    location text,
    start_at timestamp with time zone,
    end_at timestamp with time zone,
    is_all_day boolean DEFAULT false,
    recurrence_rule text,
    status text DEFAULT 'confirmed'::text,
    raw_ics text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    created_by_id bigint,
    conference_url text,
    conference_provider text,
    space_id bigint,
    source text DEFAULT ''::text NOT NULL,
    external_id text DEFAULT ''::text NOT NULL,
    color text DEFAULT ''::text NOT NULL,
    timezone text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE events_id_seq OWNED BY events.id;

CREATE TABLE external_versions (
    source text NOT NULL,
    external_id text NOT NULL,
    version bigint NOT NULL,
    updated_at timestamp with time zone
);

CREATE TABLE forfait_state (
    id bigint NOT NULL,
    document text NOT NULL,
    updated_at timestamp with time zone
);

CREATE TABLE mcp_actions (
    id text NOT NULL,
    user_id text NOT NULL,
    kind text NOT NULL,
    target text NOT NULL,
    digest text NOT NULL,
    summary text DEFAULT ''::text NOT NULL,
    client_id text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    resolved_at timestamp with time zone,
    outcome text DEFAULT ''::text NOT NULL
);

CREATE TABLE porte_identities (
    user_id bigint NOT NULL,
    provider text NOT NULL,
    subject text NOT NULL,
    password_hash text DEFAULT ''::text NOT NULL,
    access_token text DEFAULT ''::text NOT NULL,
    refresh_token text DEFAULT ''::text NOT NULL,
    token_expiry timestamp with time zone,
    roles jsonb,
    roles_synced_at timestamp with time zone,
    synced_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now()
);

CREATE TABLE porte_login_codes (
    code_hash text NOT NULL,
    user_id bigint NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    consumed_at timestamp with time zone,
    provider text DEFAULT ''::text NOT NULL,
    subject text DEFAULT ''::text NOT NULL
);

CREATE TABLE porte_sessions (
    id bigint NOT NULL,
    token_hash text NOT NULL,
    user_id bigint NOT NULL,
    label text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_used_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone
);

CREATE SEQUENCE porte_sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE porte_sessions_id_seq OWNED BY porte_sessions.id;

CREATE TABLE space_members (
    id bigint NOT NULL,
    space_id bigint NOT NULL,
    user_id bigint NOT NULL,
    role text DEFAULT 'member'::text NOT NULL,
    joined_at timestamp with time zone
);

CREATE SEQUENCE space_members_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE space_members_id_seq OWNED BY space_members.id;

CREATE TABLE spaces (
    id bigint NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    registre_space_id text
);

CREATE SEQUENCE spaces_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE spaces_id_seq OWNED BY spaces.id;

CREATE TABLE users (
    id bigint NOT NULL,
    email text,
    name text,
    avatar_url text,
    avatar_source text,
    oidc_picture_url text,
    oidc_access_token text,
    oidc_refresh_token text,
    oidc_token_expiry timestamp with time zone,
    profile_synced_at timestamp with time zone,
    password_hash text,
    created_at timestamp with time zone,
    oidc_subject text,
    avatar_upload_path text,
    locale text DEFAULT 'en'::text NOT NULL
);

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE users_id_seq OWNED BY users.id;

ALTER TABLE ONLY app_settings ALTER COLUMN id SET DEFAULT nextval('app_settings_id_seq'::regclass);

ALTER TABLE ONLY calendar_members ALTER COLUMN id SET DEFAULT nextval('calendar_members_id_seq'::regclass);

ALTER TABLE ONLY calendars ALTER COLUMN id SET DEFAULT nextval('calendars_id_seq'::regclass);

ALTER TABLE ONLY event_attendees ALTER COLUMN id SET DEFAULT nextval('event_attendees_id_seq'::regclass);

ALTER TABLE ONLY events ALTER COLUMN id SET DEFAULT nextval('events_id_seq'::regclass);

ALTER TABLE ONLY porte_sessions ALTER COLUMN id SET DEFAULT nextval('porte_sessions_id_seq'::regclass);

ALTER TABLE ONLY space_members ALTER COLUMN id SET DEFAULT nextval('space_members_id_seq'::regclass);

ALTER TABLE ONLY spaces ALTER COLUMN id SET DEFAULT nextval('spaces_id_seq'::regclass);

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);

ALTER TABLE ONLY app_settings
    ADD CONSTRAINT app_settings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY calendar_members
    ADD CONSTRAINT calendar_members_pkey PRIMARY KEY (id);

ALTER TABLE ONLY calendars
    ADD CONSTRAINT calendars_pkey PRIMARY KEY (id);

ALTER TABLE ONLY directory_deletion_state
    ADD CONSTRAINT directory_deletion_state_pkey PRIMARY KEY (id);

ALTER TABLE ONLY event_attendees
    ADD CONSTRAINT event_attendees_pkey PRIMARY KEY (id);

ALTER TABLE ONLY events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id);

ALTER TABLE ONLY external_versions
    ADD CONSTRAINT external_versions_pkey PRIMARY KEY (source, external_id);

ALTER TABLE ONLY forfait_state
    ADD CONSTRAINT forfait_state_pkey PRIMARY KEY (id);

ALTER TABLE ONLY mcp_actions
    ADD CONSTRAINT mcp_actions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY porte_identities
    ADD CONSTRAINT porte_identities_pkey PRIMARY KEY (provider, subject);

ALTER TABLE ONLY porte_login_codes
    ADD CONSTRAINT porte_login_codes_pkey PRIMARY KEY (code_hash);

ALTER TABLE ONLY porte_sessions
    ADD CONSTRAINT porte_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY porte_sessions
    ADD CONSTRAINT porte_sessions_token_hash_key UNIQUE (token_hash);

ALTER TABLE ONLY space_members
    ADD CONSTRAINT space_members_pkey PRIMARY KEY (id);

ALTER TABLE ONLY spaces
    ADD CONSTRAINT spaces_pkey PRIMARY KEY (id);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

CREATE UNIQUE INDEX idx_calendar_member ON calendar_members USING btree (calendar_id, user_id);

CREATE INDEX idx_calendars_cal_dav_path ON calendars USING btree (cal_dav_path);

CREATE UNIQUE INDEX idx_calendars_one_per_space ON calendars USING btree (space_id) WHERE (space_id IS NOT NULL);

CREATE INDEX idx_calendars_owner_id ON calendars USING btree (owner_id);

CREATE INDEX idx_calendars_slug ON calendars USING btree (slug);

CREATE INDEX idx_calendars_space_id ON calendars USING btree (space_id);

CREATE INDEX idx_event_attendees_event_id ON event_attendees USING btree (event_id);

CREATE INDEX idx_event_attendees_user_id ON event_attendees USING btree (user_id);

CREATE UNIQUE INDEX idx_event_source_external ON events USING btree (source, external_id) WHERE (source <> ''::text);

CREATE INDEX idx_events_calendar_id ON events USING btree (calendar_id);

CREATE INDEX idx_events_created_by_id ON events USING btree (created_by_id);

CREATE INDEX idx_events_end_at ON events USING btree (end_at);

CREATE INDEX idx_events_space_id ON events USING btree (space_id);

CREATE INDEX idx_events_start_at ON events USING btree (start_at);

CREATE UNIQUE INDEX idx_space_member ON space_members USING btree (space_id, user_id);

CREATE INDEX idx_space_members_space_id ON space_members USING btree (space_id);

CREATE INDEX idx_space_members_user_id ON space_members USING btree (user_id);

CREATE UNIQUE INDEX idx_spaces_registre_space_id ON spaces USING btree (registre_space_id);

CREATE UNIQUE INDEX idx_uid_calendar ON events USING btree (calendar_id, uid);

CREATE UNIQUE INDEX idx_users_email ON users USING btree (email);

CREATE UNIQUE INDEX idx_users_o_id_c_subject ON users USING btree (oidc_subject);

CREATE INDEX mcp_actions_open_idx ON mcp_actions USING btree (user_id, created_at DESC) WHERE (resolved_at IS NULL);

CREATE INDEX porte_identities_user_idx ON porte_identities USING btree (user_id);

CREATE INDEX porte_sessions_expiry_idx ON porte_sessions USING btree (expires_at) WHERE (expires_at IS NOT NULL);

CREATE INDEX porte_sessions_user_idx ON porte_sessions USING btree (user_id);

ALTER TABLE ONLY events
    ADD CONSTRAINT fk_events_created_by FOREIGN KEY (created_by_id) REFERENCES users(id);

ALTER TABLE ONLY space_members
    ADD CONSTRAINT fk_space_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY space_members
    ADD CONSTRAINT fk_spaces_members FOREIGN KEY (space_id) REFERENCES spaces(id);

ALTER TABLE ONLY porte_identities
    ADD CONSTRAINT porte_identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY porte_login_codes
    ADD CONSTRAINT porte_login_codes_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY porte_sessions
    ADD CONSTRAINT porte_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- +goose Down
SELECT 'baseline migration is not reversible' WHERE false;
