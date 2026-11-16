import { expect, test } from 'bun:test';
import type { Activity } from './backend';
import { LANE_WIDTH, dayStrips, eventGutter, spanLabel, startOfDay } from './activity-lanes';

const day = startOfDay(new Date(2026, 10, 16));
const at = (hour: number, minute = 0) => new Date(2026, 10, 16, hour, minute).toISOString();
const activity = (id: number, source: Activity['source'], from: number, to: number | null): Activity => ({
	id,
	source,
	title: '',
	start_at: at(from),
	end_at: to === null ? null : at(to),
	url: ''
});

test('an event with no strip during it keeps only the edge', () => {
	const strips = dayStrips([activity(1, 'sablier', 9, 10)], day, day);
	expect(eventGutter(strips, at(14), at(15))).toBe(2);
	expect(eventGutter(strips, at(10), at(11))).toBe(2);
});

test('an event leaves room for the lanes that run during it', () => {
	const strips = dayStrips([activity(1, 'sablier', 9, 10), activity(2, 'echo', 13, 14)], day, day);
	expect(eventGutter(strips, at(9, 30), at(11))).toBe(2 + LANE_WIDTH);
	expect(eventGutter(strips, at(13), at(14))).toBe(2 + 2 * LANE_WIDTH);
});

test('a day with one source uses the first lane', () => {
	const strips = dayStrips([activity(2, 'echo', 13, 14)], day, day);
	expect(strips[0].lane).toBe(0);
	expect(eventGutter(strips, at(13), at(14))).toBe(2 + LANE_WIDTH);
});

test('a running activity ends now and another day is left out', () => {
	const now = new Date(2026, 10, 16, 11).getTime();
	const strips = dayStrips([activity(1, 'sablier', 10, null), { ...activity(3, 'echo', 9, 10), start_at: new Date(2026, 10, 17, 9).toISOString(), end_at: new Date(2026, 10, 17, 10).toISOString() }], day, now);
	expect(strips).toHaveLength(1);
	expect(strips[0].to).toBe(now);
});

test('spans read as minutes and hours', () => {
	expect(spanLabel(45 * 60000)).toBe('45 min');
	expect(spanLabel(120 * 60000)).toBe('2 h');
	expect(spanLabel(90 * 60000)).toBe('1 h 30');
	expect(spanLabel(65 * 60000)).toBe('1 h 05');
});
