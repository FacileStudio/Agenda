import { describe, expect, test } from 'bun:test';
import type { Plan, PlanBlock } from '$lib/backend';
import { ASK_LIMIT, PROMPT_LIMIT, blocksOnDay, conflictIds, dayKey, openBlocks, pageContext, planPrompt, planSince } from './bulle';

const range = { from: '2026-11-16', to: '2026-11-22', timezone: 'Europe/Paris', now: new Date('2026-11-16T08:00:00Z') };

const block = (id: number, start: string, eventId: number | null = null): PlanBlock => ({
	id, task_ref: '', task_url: '', title: 'Focus', start_at: start, end_at: start, calendar_id: 1, reason: '', event_id: eventId
});

const plan = (over: Partial<Plan> = {}): Plan => ({
	id: 1, space_id: null, title: 'Week', timezone: 'UTC', state: 'proposed', expired: false, warnings: [],
	blocks: [], created_at: '2026-11-16T08:05:00Z', expires_at: '2026-11-23T08:05:00Z', ...over
});

describe('planPrompt', () => {
	test('names the range, the zone, the tools and the one-draft rule', () => {
		const text = planPrompt('week', range);
		for (const part of ['2026-11-16 to 2026-11-22', 'Europe/Paris', 'opus_search_tasks', 'agenda_find_free_slots', 'ONE draft', 'Never create']) {
			expect(text).toContain(part);
		}
		expect(text).not.toContain("The user's request");
	});

	test('a single day reads as one date', () => {
		expect(planPrompt('today', { ...range, to: range.from })).toContain('the user\'s day: 2026-11-16,');
	});

	test('the user\'s words come last and the prompt stays under the cap', () => {
		const text = planPrompt('ask', range, 'x'.repeat(5000));
		expect(text.length).toBeLessThanOrEqual(PROMPT_LIMIT);
		expect(text).toContain("The user's request: " + 'x'.repeat(ASK_LIMIT));
		expect(text.endsWith('x'.repeat(ASK_LIMIT))).toBe(true);
	});

	test('the fixed template alone fits in the cap with room for an ask', () => {
		expect(planPrompt('week', range).length + ASK_LIMIT + 30).toBeLessThan(PROMPT_LIMIT);
	});
});

describe('pageContext', () => {
	test('is dropped off https, since Bulle refuses other urls', () => {
		expect(pageContext('week', 'Week', 'http://localhost:5173')).toBeUndefined();
		expect(pageContext('week', 'Week', 'https://agenda.facile.studio')).toEqual({
			app: 'agenda', id: 'calendar-week', title: 'Week', url: 'https://agenda.facile.studio/calendar'
		});
	});
});

describe('plans', () => {
	test('planSince keeps the newest proposed plan the run made', () => {
		const old = plan({ id: 1, created_at: '2026-11-16T07:00:00Z' });
		const mine = plan({ id: 2, created_at: '2026-11-16T08:10:00Z' });
		const later = plan({ id: 3, created_at: '2026-11-16T08:20:00Z' });
		expect(planSince([old, mine, later], new Date('2026-11-16T08:00:00Z'))?.id).toBe(3);
		expect(planSince([old], new Date('2026-11-16T08:00:00Z'))).toBeNull();
		expect(planSince([plan({ expired: true })], new Date('2026-11-16T08:00:00Z'))).toBeNull();
	});

	test('openBlocks leaves out applied blocks and closed plans', () => {
		const p = plan({ blocks: [block(1, '2026-11-17T09:00:00Z'), block(2, '2026-11-17T11:00:00Z', 9)] });
		expect(openBlocks(p).map((b) => b.id)).toEqual([1]);
		expect(openBlocks({ ...p, state: 'applied' })).toEqual([]);
		expect(openBlocks(null)).toEqual([]);
	});

	test('blocksOnDay picks the local day, earliest first', () => {
		const p = plan({ blocks: [block(2, new Date(2026, 10, 17, 14).toISOString()), block(1, new Date(2026, 10, 17, 9).toISOString()), block(3, new Date(2026, 10, 18, 9).toISOString())] });
		expect(blocksOnDay(p, new Date(2026, 10, 17)).map((b) => b.id)).toEqual([1, 2]);
	});

	test('conflictIds reads the ids of the 409 message', () => {
		expect(conflictIds('Blocks now overlapping an event (block ids 12, 13). Apply again with force to keep them, or leave them out')).toEqual([12, 13]);
		expect(conflictIds('something else')).toEqual([]);
	});

	test('dayKey is the local date', () => {
		expect(dayKey(new Date(2026, 10, 7))).toBe('2026-11-07');
	});
});
