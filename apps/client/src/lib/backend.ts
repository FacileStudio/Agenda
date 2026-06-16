const backendBaseUrl = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') || '';

export type AuthResponse = {
	user_id: string;
	token: string;
};

export type UserProfile = {
	id: string;
	email: string;
	name: string;
	avatar_url: string;
	avatar_source: string;
	created_at: string;
};

export type MeResponse = {
	user: UserProfile;
};

export type CalendarItem = {
	id: number;
	owner_id: number;
	name: string;
	color: string;
	description: string;
	is_personal: boolean;
	role: string;
};

export type CalendarMember = {
	user_id: number;
	email: string;
	name: string;
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
	error?: { message?: string };
};

export class ApiError extends Error {
	status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

async function apiFetch<T>(path: string, options: RequestInit = {}) {
	const headers = new Headers(options.headers);
	if (!headers.has('Content-Type') && options.body) {
		headers.set('Content-Type', 'application/json');
	}
	const response = await fetch(`${backendBaseUrl}${path}`, { ...options, headers, credentials: 'include' });
	if (!response.ok) {
		let payload: ApiErrorPayload | undefined;
		try {
			payload = (await response.json()) as ApiErrorPayload;
		} catch {
			payload = undefined;
		}
		throw new ApiError(response.status, payload?.error?.message || `Request failed with status ${response.status}`);
	}
	return (await response.json()) as T;
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

	register(email: string, password: string) {
		return apiFetch<AuthResponse>('/auth/register', {
			method: 'POST',
			body: JSON.stringify({ email, password })
		});
	},
	login(email: string, password: string) {
		return apiFetch<AuthResponse>('/auth/login', {
			method: 'POST',
			body: JSON.stringify({ email, password })
		});
	},
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

	listCalendars() {
		return apiFetch<CalendarItem[]>('/calendars/');
	},
	createCalendar(data: { name: string; color: string; description?: string }) {
		return apiFetch<CalendarItem>('/calendars/', {
			method: 'POST',
			body: JSON.stringify(data)
		});
	},
	getCalendar(id: number) {
		return apiFetch<CalendarItem>(`/calendars/${id}/`);
	},
	updateCalendar(id: number, data: { name: string; color: string; description?: string }) {
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
	}
};
