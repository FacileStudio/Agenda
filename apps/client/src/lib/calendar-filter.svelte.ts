import { browser } from '$app/environment';
import type { CalendarItem } from '$lib/backend';
import { compareNames } from '$lib/i18n/format';

const STORAGE_KEY = 'agenda:calendar-filter';

export type CalendarFilter = 'all' | number;

function load(): CalendarFilter {
	if (!browser) return 'all';
	try {
		const id = Number(localStorage.getItem(STORAGE_KEY));
		return Number.isInteger(id) && id > 0 ? id : 'all';
	} catch {
		return 'all';
	}
}

let current = $state<CalendarFilter>(load());

/** The calendar the views are narrowed to, or 'all'. */
export function calendarFilter(): CalendarFilter {
	return current;
}

/** Narrows the views to one calendar, or widens them back with 'all'. */
export function setCalendarFilter(next: CalendarFilter) {
	current = next;
	if (!browser) return;
	try {
		if (next === 'all') localStorage.removeItem(STORAGE_KEY);
		else localStorage.setItem(STORAGE_KEY, String(next));
	} catch {}
}

/** The calendars the filter lets through; a filter on a calendar that is gone shows them all. */
export function visibleCalendars(calendars: CalendarItem[]): CalendarItem[] {
	const id = current;
	if (id === 'all' || !calendars.some((c) => c.id === id)) return calendars;
	return calendars.filter((c) => c.id === id);
}

/** Space calendars first, then personal ones, each by name. */
export function sortCalendars(calendars: CalendarItem[]): CalendarItem[] {
	return [...calendars].sort(
		(a, b) => Number(!a.space_id) - Number(!b.space_id) || compareNames(a.name, b.name)
	);
}
