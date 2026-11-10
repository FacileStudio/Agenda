CREATE TABLE public.antenne_outbox (
    id bigint NOT NULL,
    channel text NOT NULL,
    payload jsonb NOT NULL,
    idempotency_key text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone
);

CREATE SEQUENCE public.antenne_outbox_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.antenne_outbox_id_seq OWNED BY public.antenne_outbox.id;

CREATE TABLE public.app_settings (
    id bigint NOT NULL,
    encryption_key text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE public.app_settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.app_settings_id_seq OWNED BY public.app_settings.id;

CREATE TABLE public.calendar_members (
    id bigint NOT NULL,
    calendar_id bigint,
    user_id bigint,
    role text,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE public.calendar_members_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.calendar_members_id_seq OWNED BY public.calendar_members.id;

CREATE TABLE public.calendars (
    id bigint NOT NULL,
    owner_id bigint,
    slug text,
    name text,
    color text,
    description text,
    is_personal boolean DEFAULT false,
    sync_token text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    echo_url text,
    space_id bigint,
    cal_dav_path text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE public.calendars_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.calendars_id_seq OWNED BY public.calendars.id;

CREATE TABLE public.directory_deletion_state (
    id smallint NOT NULL,
    state bytea NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT directory_deletion_state_id_check CHECK ((id = 1))
);

CREATE TABLE public.event_attendees (
    id bigint NOT NULL,
    event_id bigint,
    user_id bigint,
    email text,
    response text DEFAULT 'needs-action'::text,
    created_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE public.event_attendees_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.event_attendees_id_seq OWNED BY public.event_attendees.id;

CREATE TABLE public.events (
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
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    created_by_id bigint,
    conference_url text,
    conference_provider text,
    space_id bigint,
    source text DEFAULT ''::text NOT NULL,
    external_id text DEFAULT ''::text NOT NULL,
    color text DEFAULT ''::text NOT NULL,
    timezone text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE public.events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.events_id_seq OWNED BY public.events.id;

CREATE TABLE public.external_versions (
    source text NOT NULL,
    external_id text NOT NULL,
    version bigint NOT NULL,
    updated_at timestamp with time zone DEFAULT now()
);

CREATE TABLE public.forfait_state (
    id bigint NOT NULL,
    document text NOT NULL,
    updated_at timestamp with time zone DEFAULT now()
);

CREATE TABLE public.mcp_actions (
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

CREATE TABLE public.porte_identities (
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

CREATE TABLE public.porte_login_codes (
    code_hash text NOT NULL,
    user_id bigint NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    consumed_at timestamp with time zone,
    provider text DEFAULT ''::text NOT NULL,
    subject text DEFAULT ''::text NOT NULL
);

CREATE TABLE public.porte_sessions (
    id bigint NOT NULL,
    token_hash text NOT NULL,
    user_id bigint NOT NULL,
    label text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_used_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone
);

CREATE SEQUENCE public.porte_sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.porte_sessions_id_seq OWNED BY public.porte_sessions.id;

CREATE TABLE public.space_members (
    id bigint NOT NULL,
    space_id bigint NOT NULL,
    user_id bigint NOT NULL,
    role text DEFAULT 'member'::text NOT NULL,
    joined_at timestamp with time zone DEFAULT now()
);

CREATE SEQUENCE public.space_members_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.space_members_id_seq OWNED BY public.space_members.id;

CREATE TABLE public.spaces (
    id bigint NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    registre_space_id text
);

CREATE SEQUENCE public.spaces_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.spaces_id_seq OWNED BY public.spaces.id;

CREATE TABLE public.users (
    id bigint NOT NULL,
    email text,
    name text,
    avatar_url text DEFAULT ''::text,
    avatar_source text DEFAULT ''::text,
    oidc_picture_url text,
    oidc_access_token text DEFAULT ''::text,
    oidc_refresh_token text DEFAULT ''::text,
    oidc_token_expiry timestamp with time zone DEFAULT '0001-01-01 00:00:00+00'::timestamp with time zone,
    profile_synced_at timestamp with time zone DEFAULT '0001-01-01 00:00:00+00'::timestamp with time zone,
    password_hash text DEFAULT ''::text,
    created_at timestamp with time zone DEFAULT now(),
    oidc_subject text,
    avatar_upload_path text DEFAULT ''::text,
    locale text DEFAULT 'en'::text NOT NULL
);

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

ALTER TABLE ONLY public.antenne_outbox ALTER COLUMN id SET DEFAULT nextval('public.antenne_outbox_id_seq'::regclass);

ALTER TABLE ONLY public.app_settings ALTER COLUMN id SET DEFAULT nextval('public.app_settings_id_seq'::regclass);

ALTER TABLE ONLY public.calendar_members ALTER COLUMN id SET DEFAULT nextval('public.calendar_members_id_seq'::regclass);

ALTER TABLE ONLY public.calendars ALTER COLUMN id SET DEFAULT nextval('public.calendars_id_seq'::regclass);

ALTER TABLE ONLY public.event_attendees ALTER COLUMN id SET DEFAULT nextval('public.event_attendees_id_seq'::regclass);

ALTER TABLE ONLY public.events ALTER COLUMN id SET DEFAULT nextval('public.events_id_seq'::regclass);

ALTER TABLE ONLY public.porte_sessions ALTER COLUMN id SET DEFAULT nextval('public.porte_sessions_id_seq'::regclass);

ALTER TABLE ONLY public.space_members ALTER COLUMN id SET DEFAULT nextval('public.space_members_id_seq'::regclass);

ALTER TABLE ONLY public.spaces ALTER COLUMN id SET DEFAULT nextval('public.spaces_id_seq'::regclass);

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);

ALTER TABLE ONLY public.antenne_outbox
    ADD CONSTRAINT antenne_outbox_idempotency_key_key UNIQUE (idempotency_key);

ALTER TABLE ONLY public.antenne_outbox
    ADD CONSTRAINT antenne_outbox_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.app_settings
    ADD CONSTRAINT app_settings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.calendar_members
    ADD CONSTRAINT calendar_members_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.calendars
    ADD CONSTRAINT calendars_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.directory_deletion_state
    ADD CONSTRAINT directory_deletion_state_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.event_attendees
    ADD CONSTRAINT event_attendees_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.external_versions
    ADD CONSTRAINT external_versions_pkey PRIMARY KEY (source, external_id);

ALTER TABLE ONLY public.forfait_state
    ADD CONSTRAINT forfait_state_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.mcp_actions
    ADD CONSTRAINT mcp_actions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.porte_identities
    ADD CONSTRAINT porte_identities_pkey PRIMARY KEY (provider, subject);

ALTER TABLE ONLY public.porte_login_codes
    ADD CONSTRAINT porte_login_codes_pkey PRIMARY KEY (code_hash);

ALTER TABLE ONLY public.porte_sessions
    ADD CONSTRAINT porte_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.porte_sessions
    ADD CONSTRAINT porte_sessions_token_hash_key UNIQUE (token_hash);

ALTER TABLE ONLY public.space_members
    ADD CONSTRAINT space_members_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.spaces
    ADD CONSTRAINT spaces_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

CREATE INDEX antenne_outbox_sent ON public.antenne_outbox USING btree (sent_at) WHERE (sent_at IS NOT NULL);

CREATE INDEX antenne_outbox_unsent ON public.antenne_outbox USING btree (id) WHERE (sent_at IS NULL);

CREATE UNIQUE INDEX idx_calendar_member ON public.calendar_members USING btree (calendar_id, user_id);

CREATE INDEX idx_calendars_cal_dav_path ON public.calendars USING btree (cal_dav_path);

CREATE UNIQUE INDEX idx_calendars_one_per_space ON public.calendars USING btree (space_id) WHERE (space_id IS NOT NULL);

CREATE INDEX idx_calendars_owner_id ON public.calendars USING btree (owner_id);

CREATE INDEX idx_calendars_slug ON public.calendars USING btree (slug);

CREATE INDEX idx_calendars_space_id ON public.calendars USING btree (space_id);

CREATE INDEX idx_event_attendees_event_id ON public.event_attendees USING btree (event_id);

CREATE INDEX idx_event_attendees_user_id ON public.event_attendees USING btree (user_id);

CREATE UNIQUE INDEX idx_event_source_external ON public.events USING btree (source, external_id) WHERE (source <> ''::text);

CREATE INDEX idx_events_calendar_id ON public.events USING btree (calendar_id);

CREATE INDEX idx_events_created_by_id ON public.events USING btree (created_by_id);

CREATE INDEX idx_events_end_at ON public.events USING btree (end_at);

CREATE INDEX idx_events_space_id ON public.events USING btree (space_id);

CREATE INDEX idx_events_start_at ON public.events USING btree (start_at);

CREATE UNIQUE INDEX idx_space_member ON public.space_members USING btree (space_id, user_id);

CREATE INDEX idx_space_members_space_id ON public.space_members USING btree (space_id);

CREATE INDEX idx_space_members_user_id ON public.space_members USING btree (user_id);

CREATE UNIQUE INDEX idx_spaces_registre_space_id ON public.spaces USING btree (registre_space_id);

CREATE UNIQUE INDEX idx_uid_calendar ON public.events USING btree (calendar_id, uid);

CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);

CREATE UNIQUE INDEX idx_users_o_id_c_subject ON public.users USING btree (oidc_subject);

CREATE INDEX mcp_actions_open_idx ON public.mcp_actions USING btree (user_id, created_at DESC) WHERE (resolved_at IS NULL);

CREATE INDEX porte_identities_user_idx ON public.porte_identities USING btree (user_id);

CREATE INDEX porte_sessions_expiry_idx ON public.porte_sessions USING btree (expires_at) WHERE (expires_at IS NOT NULL);

CREATE INDEX porte_sessions_user_idx ON public.porte_sessions USING btree (user_id);

ALTER TABLE ONLY public.events
    ADD CONSTRAINT fk_events_created_by FOREIGN KEY (created_by_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.space_members
    ADD CONSTRAINT fk_space_members_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.space_members
    ADD CONSTRAINT fk_spaces_members FOREIGN KEY (space_id) REFERENCES public.spaces(id);

ALTER TABLE ONLY public.porte_identities
    ADD CONSTRAINT porte_identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.porte_login_codes
    ADD CONSTRAINT porte_login_codes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.porte_sessions
    ADD CONSTRAINT porte_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

