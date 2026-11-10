const backendBaseUrl = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') || '';
const apiBaseUrl = `${backendBaseUrl}/api`;

export type UserProfile = {
	id: string;
	email: string;
	name: string;
	avatar_url: string;
	avatar_source: string;
	locale: string;
	created_at: string;
};

export type MeResponse = {
	user: UserProfile;
};

export type CalendarItem = {
	id: number;
	owner_id: number;
	space_id: number | null;
	name: string;
	color: string;
	description: string;
	echo_url: string;
	is_personal: boolean;
	role: string;
};

export type CalendarMember = {
	user_id: number;
	email: string;
	name: string;
	avatar_url: string;
	role: string;
};

export type EventCreator = {
	id: number;
	name: string;
	email: string;
	avatar_url: string;
};

export type AgendaEvent = {
	id: number;
	calendar_id: number;
	uid: string;
	etag: string;
	title: string;
	description: string;
	location: string;
	start_at: string;
	end_at: string;
	is_all_day: boolean;
	recurrence_rule: string;
	status: string;
	conference_url: string;
	conference_provider: string;
	created_by: EventCreator | null;
	created_at: string;
	updated_at: string;
};

export type CreateEventRequest = {
	title: string;
	description?: string;
	location?: string;
	start_at: string;
	end_at: string;
	is_all_day?: boolean;
	recurrence_rule?: string;
	status?: string;
	conference_url?: string;
	conference_provider?: string;
	timezone?: string;
};

export type SpaceItem = {
	id: number;
	name: string;
	description: string;
	registre_space_id: string;
	role: string;
	created_at: string;
	updated_at: string;
};

export type SpaceMember = {
	user_id: number;
	email: string;
	name: string;
	avatar_url: string;
	role: string;
	joined_at: string;
};

export type ApiTokenStatus = {
	has_token: boolean;
	name?: string;
	created_at?: string;
};

export type ApiTokenCreated = {
	token: string;
	name: string;
	created_at: string;
};

type ApiErrorPayload = {
	error?: { message?: string; upgrade_url?: string };
};

/** The event a pending action would delete, as it is now. */
export type PendingEvent = {
	id: number;
	calendar_id: number;
	title: string;
	start_at: string;
	end_at: string;
	is_all_day: boolean;
	location: string;
	recurring: boolean;
};

/** One action an MCP client asked for, waiting for the person. */
export type PendingAction = {
	id: string;
	kind: string;
	client_id: string;
	summary: string;
	created_at: string;
	expires_at: string;
	state: 'ready' | 'changed' | 'missing';
	event?: PendingEvent;
};

export class ApiError extends Error {
	status: number;
	upgradeUrl?: string;

	constructor(status: number, message: string, upgradeUrl?: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.upgradeUrl = upgradeUrl;
	}
}

export type BulleIdea = {
	id: string;
	kind: string;
	title: string;
	why: string;
	draft: string;
	sources: { label: string; ref: string }[] | null;
	status: string;
	expires_at: string;
};

export type IdeasResponse = {
	ai_access: boolean;
	upgrade_url?: string;
	ideas: BulleIdea[];
	generating: boolean;
	last_batch_at: string | null;
};

export type IdeaRunResponse = {
	id: string;
	state: 'running' | 'needs_approval' | 'done' | 'failed';
	summary: string;
	outputs: { call_id: string; tool: string; url: string; preview: string }[] | null;
	pending_approvals: { call_id: string; tool: string; args: unknown }[] | null;
	continue_url: string;
};

// Methods that cannot change anything, and so do not need the CSRF header.
export const REGISTRE_SPACES_URL = 'https://sso.facile.studio/spaces';

/**
 * Where a space is managed. Registre owns every shared space's name, members and roles;
 * Agenda only shows them, so each edit button links here instead.
 */
export function registreSpaceUrl(registreSpaceId = ''): string {
	return registreSpaceId ? `${REGISTRE_SPACES_URL}/${encodeURIComponent(registreSpaceId)}` : REGISTRE_SPACES_URL;
}

const SAFE_METHODS = new Set(['GET', 'HEAD', 'OPTIONS']);

// porte authenticates the cookie in preference to the Authorization header,
// and refuses a cookie-authenticated *mutating* request that does not carry
// X-Facile-CSRF. Any non-empty value counts: the header's presence is the whole
// signal, because a browser will not attach a custom header to a cross-site
// form post. Without this every write returns 403 while every read succeeds,
// which is a failure mode that looks like "the app is fine, saving is broken".
function withCSRF(headers: Headers, method: string): Headers {
	if (!SAFE_METHODS.has(method.toUpperCase())) {
		headers.set('X-Facile-CSRF', '1');
	}
	return headers;
}

const spaceQuery = (spaceId: number | null): string => (spaceId === null ? '' : `?space_id=${spaceId}`);

async function apiFetch<T>(path: string, options: RequestInit = {}) {
	const headers = withCSRF(new Headers(options.headers), options.method ?? 'GET');
	if (!headers.has('Content-Type') && options.body) {
		headers.set('Content-Type', 'application/json');
	}
	const response = await fetch(`${apiBaseUrl}${path}`, { ...options, headers, credentials: 'include' });
	if (!response.ok) {
		let payload: ApiErrorPayload | undefined;
		try {
			payload = (await response.json()) as ApiErrorPayload;
		} catch {
			payload = undefined;
		}
		throw new ApiError(
			response.status,
			payload?.error?.message || `Request failed with status ${response.status}`,
			response.status === 402 ? payload?.error?.upgrade_url : undefined
		);
	}
	const text = await response.text();
	return (text ? JSON.parse(text) : {}) as T;
}

export function resolveFileUrl(path: string) {
	if (!path) return '';
	if (/^https?:\/\//.test(path)) return path;
	return `${backendBaseUrl}${path.startsWith('/') ? path : `/${path}`}`;
}

function normalizeUser(user: UserProfile): UserProfile {
	return { ...user, avatar_url: resolveFileUrl(user.avatar_url) };
}

export const backend = {
	baseUrl: backendBaseUrl,
	apiBaseUrl,

	logout() {
		return apiFetch<{ ok: boolean }>('/auth/logout', { method: 'POST' });
	},
	me() {
		return apiFetch<MeResponse>('/users/me').then((r) => ({
			user: normalizeUser(r.user)
		}));
	},
	syncProfile() {
		return apiFetch<{ synced: boolean }>('/auth/sync-profile', { method: 'POST' });
	},

	listPending() {
		return apiFetch<{ actions: PendingAction[] }>('/mcp/pending');
	},
	confirmPending(id: string) {
		return apiFetch<{ status: string }>(`/mcp/pending/${encodeURIComponent(id)}/confirm`, { method: 'POST' });
	},
	dismissPending(id: string) {
		return apiFetch<{ status: string }>(`/mcp/pending/${encodeURIComponent(id)}/dismiss`, { method: 'POST' });
	},

	listIdeas(spaceId: number | null) {
		return apiFetch<IdeasResponse>(`/ideas${spaceQuery(spaceId)}`);
	},
	generateIdeas(spaceId: number | null) {
		return apiFetch<{ batch_id: string }>(`/ideas/generate${spaceQuery(spaceId)}`, { method: 'POST' });
	},
	runIdea(id: string, spaceId: number | null) {
		return apiFetch<{ run_id: string }>(`/ideas/${encodeURIComponent(id)}/run${spaceQuery(spaceId)}`, { method: 'POST' });
	},
	dismissIdea(id: string) {
		return apiFetch<object>(`/ideas/${encodeURIComponent(id)}/dismiss`, { method: 'POST' });
	},
	ideaRun(runId: string) {
		return apiFetch<IdeaRunResponse>(`/ideas/runs/${encodeURIComponent(runId)}`);
	},
	decideIdea(runId: string, callId: string, approve: boolean) {
		return apiFetch<object>(`/ideas/runs/${encodeURIComponent(runId)}/approvals/${encodeURIComponent(callId)}`, {
			method: 'POST',
			body: JSON.stringify({ approve })
		});
	},

	listCalendars(spaceId?: number) {
		const params = new URLSearchParams();
		if (spaceId != null) params.set('space_id', String(spaceId));
		const qs = params.toString();
		return apiFetch<CalendarItem[]>(`/calendars/${qs ? `?${qs}` : ''}`);
	},
	createCalendar(data: { name: string; color: string; description?: string; echo_url?: string }) {
		return apiFetch<CalendarItem>('/calendars/', {
			method: 'POST',
			body: JSON.stringify(data)
		});
	},
	getCalendar(id: number) {
		return apiFetch<CalendarItem>(`/calendars/${id}/`);
	},
	updateCalendar(id: number, data: { name: string; color: string; description?: string; echo_url?: string }) {
		return apiFetch<CalendarItem>(`/calendars/${id}/`, {
			method: 'PUT',
			body: JSON.stringify(data)
		});
	},
	deleteCalendar(id: number) {
		return apiFetch<{ ok: boolean }>(`/calendars/${id}/`, { method: 'DELETE' });
	},
	listMembers(calendarId: number) {
		return apiFetch<CalendarMember[]>(`/calendars/${calendarId}/members`);
	},
	shareCalendar(calendarId: number, email: string, role: string) {
		return apiFetch<{ ok: boolean }>(`/calendars/${calendarId}/members`, {
			method: 'POST',
			body: JSON.stringify({ email, role })
		});
	},
	removeMember(calendarId: number, userId: number) {
		return apiFetch<{ ok: boolean }>(`/calendars/${calendarId}/members/${userId}`, {
			method: 'DELETE'
		});
	},

	listEvents(calendarId: number, from?: string, to?: string) {
		const params = new URLSearchParams();
		if (from) params.set('from', from);
		if (to) params.set('to', to);
		params.set('tz', Intl.DateTimeFormat().resolvedOptions().timeZone);
		const qs = params.toString();
		return apiFetch<AgendaEvent[]>(`/calendars/${calendarId}/events/${qs ? `?${qs}` : ''}`);
	},
	createEvent(calendarId: number, data: CreateEventRequest) {
		return apiFetch<AgendaEvent>(`/calendars/${calendarId}/events/`, {
			method: 'POST',
			body: JSON.stringify(data)
		});
	},
	getEvent(eventId: number) {
		return apiFetch<AgendaEvent>(`/events/${eventId}/`);
	},
	updateEvent(eventId: number, calendarId: number, data: Partial<CreateEventRequest>) {
		// calendar_id lets the API move the event to another calendar.
		return apiFetch<AgendaEvent>(`/events/${eventId}/`, {
			method: 'PUT',
			body: JSON.stringify({ ...data, calendar_id: calendarId })
		});
	},
	deleteEvent(eventId: number) {
		return apiFetch<{ ok: boolean }>(`/events/${eventId}/`, { method: 'DELETE' });
	},

	getApiToken() {
		return apiFetch<ApiTokenStatus>('/users/me/api-token');
	},
	createApiToken(name: string) {
		return apiFetch<ApiTokenCreated>('/users/me/api-token', {
			method: 'POST',
			body: JSON.stringify({ name })
		});
	},
	deleteApiToken() {
		return apiFetch<{ deleted: boolean }>('/users/me/api-token', { method: 'DELETE' });
	},

	listSpaces() {
		return apiFetch<SpaceItem[]>('/spaces/');
	},
	getSpace(id: number) {
		return apiFetch<SpaceItem>(`/spaces/${id}/`);
	},
	listSpaceMembers(spaceId: number) {
		return apiFetch<SpaceMember[]>(`/spaces/${spaceId}/members`);
	},
};
