export const SNAP_MINUTES = 15;
export const DAY_MINUTES = 1440;
export const DRAG_THRESHOLD_PX = 4;

export type MinuteRange = { start: number; end: number };

export function snapNearest(minutes: number): number {
	return Math.round(minutes / SNAP_MINUTES) * SNAP_MINUTES;
}

function clampRange(start: number, end: number): MinuteRange {
	const length = end - start;
	const s = Math.min(Math.max(start, 0), DAY_MINUTES - length);
	return { start: s, end: s + length };
}

export function moveRange(range: MinuteRange, deltaMinutes: number): MinuteRange {
	return clampRange(range.start + snapNearest(deltaMinutes), range.end + snapNearest(deltaMinutes));
}

export function resizeRange(range: MinuteRange, edge: 'top' | 'bottom', deltaMinutes: number): MinuteRange {
	const delta = snapNearest(deltaMinutes);
	if (edge === 'top') {
		const start = Math.min(Math.max(range.start + delta, 0), range.end - SNAP_MINUTES);
		return { start, end: range.end };
	}
	const end = Math.max(Math.min(range.end + delta, DAY_MINUTES), range.start + SNAP_MINUTES);
	return { start: range.start, end };
}

export function drawRange(anchorMinutes: number, currentMinutes: number): MinuteRange {
	const low = Math.min(anchorMinutes, currentMinutes);
	const high = Math.max(anchorMinutes, currentMinutes);
	let start = Math.floor(low / SNAP_MINUTES) * SNAP_MINUTES;
	let end = Math.ceil(high / SNAP_MINUTES) * SNAP_MINUTES;
	if (end - start < SNAP_MINUTES) end = start + SNAP_MINUTES;
	if (end > DAY_MINUTES) {
		end = DAY_MINUTES;
		start = Math.min(start, end - SNAP_MINUTES);
	}
	return { start: Math.max(start, 0), end };
}

export function minutesOfDay(date: Date): number {
	return date.getHours() * 60 + date.getMinutes();
}

export function atMinutes(day: Date, minutes: number): Date {
	return new Date(day.getFullYear(), day.getMonth(), day.getDate(), 0, minutes, 0, 0);
}

export function formatMinutes(minutes: number): string {
	const h = Math.floor(minutes / 60) % 24;
	const m = minutes % 60;
	return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

export function toLocalInput(date: Date): string {
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}
