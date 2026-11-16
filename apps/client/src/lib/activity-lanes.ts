import type { Activity } from './backend';

export const LANE_WIDTH = 10;
const EDGE = 2;
const DAY_MS = 86400000;
const MIN_EVENT_MS = 30 * 60000;
const ORDER: Activity['source'][] = ['sablier', 'echo'];

export type Strip = { activity: Activity; from: number; to: number; lane: number };

/** Local midnight of a date, in milliseconds. */
export function startOfDay(day: Date): number {
	return new Date(day.getFullYear(), day.getMonth(), day.getDate()).getTime();
}

/** The activities that fall on a day, clipped to it. Lanes count from the right edge, and a day with one source uses one lane. */
export function dayStrips(activities: Activity[], dayStart: number, now: number): Strip[] {
	const clipped = activities.flatMap((activity) => {
		const start = new Date(activity.start_at).getTime();
		const end = activity.end_at ? new Date(activity.end_at).getTime() : Math.max(now, start);
		const from = Math.max(start, dayStart);
		const to = Math.min(end, dayStart + DAY_MS);
		return to > from ? [{ activity, from, to }] : [];
	});
	const sources = ORDER.filter((source) => clipped.some((strip) => strip.activity.source === source));
	return clipped.map((strip) => ({ ...strip, lane: sources.indexOf(strip.activity.source) }));
}

/** The space in px an event leaves at its right edge: only the lanes whose strips run during it. */
export function eventGutter(strips: Strip[], startAt: string, endAt: string): number {
	const start = new Date(startAt).getTime();
	const end = Math.max(new Date(endAt).getTime(), start + MIN_EVENT_MS);
	const lanes = strips.filter((strip) => strip.from < end && strip.to > start).map((strip) => strip.lane);
	return EDGE + (lanes.length > 0 ? (Math.max(...lanes) + 1) * LANE_WIDTH : 0);
}

/** A span as "45 min", "2 h" or "1 h 30". */
export function spanLabel(ms: number): string {
	const minutes = Math.max(1, Math.round(ms / 60000));
	const hours = Math.floor(minutes / 60);
	const rest = minutes % 60;
	if (hours === 0) return `${rest} min`;
	return rest === 0 ? `${hours} h` : `${hours} h ${String(rest).padStart(2, '0')}`;
}
