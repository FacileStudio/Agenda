import { describe, expect, test } from 'bun:test';
import { ApiError, type BulleIdea, type IdeaRunResponse } from './backend';
import { errorKey, isActive, runOf, safeHref, toIdea, toolLabel } from './ideas';

const t = (key: string, params?: Record<string, string | number>) =>
	params ? `${key}:${Object.values(params).join(',')}` : key;

const view = (overrides: Partial<IdeaRunResponse>): IdeaRunResponse => ({
	id: 'r1',
	state: 'running',
	summary: '',
	outputs: null,
	pending_approvals: null,
	continue_url: '',
	...overrides
});

describe('safeHref', () => {
	test('keeps web links and drops everything else', () => {
		expect(safeHref('https://agenda.facile.studio/calendar')).toBe('https://agenda.facile.studio/calendar');
		expect(safeHref('agenda:5/event:812')).toBeUndefined();
		expect(safeHref('javascript:alert(1)')).toBeUndefined();
		expect(safeHref(undefined)).toBeUndefined();
	});
});

describe('runOf', () => {
	test('is running until the first poll', () => {
		expect(runOf({ runId: 'r1', view: null, error: null }, t).state).toBe('running');
	});

	test('turns a pending call into an approval with its tool label', () => {
		const run = runOf(
			{ runId: 'r1', view: view({ state: 'needs_approval', pending_approvals: [{ call_id: 'c1', tool: 'agenda_create_event', args: { title: 'Standup' } }] }), error: null },
			t
		);
		expect(run.state).toBe('needs_approval');
		expect(run.approval).toEqual({ id: 'c1', tool: 'ideas.tool.agenda_create_event', summary: 'ideas.approval.summary:ideas.tool.agenda_create_event', args: { title: 'Standup' } });
	});

	test('keeps only web links among the outputs', () => {
		const run = runOf(
			{ runId: 'r1', view: view({ state: 'done', outputs: [{ call_id: 'c', tool: 'agenda_create_event', url: 'https://agenda.facile.studio/calendar/9', preview: 'Standup' }, { call_id: 'd', tool: 'agenda_create_event', url: 'file:///x', preview: 'b' }] }), error: null },
			t
		);
		expect(run.outputs).toEqual([{ label: 'Standup', href: 'https://agenda.facile.studio/calendar/9' }]);
	});

	test('a failed run shows its reason', () => {
		expect(runOf({ runId: 'r1', view: view({ state: 'failed', summary: 'boom' }), error: null }, t)).toEqual({ state: 'failed', error: 'boom' });
	});
});

describe('isActive', () => {
	test('polls until the run is done or failed', () => {
		expect(isActive(undefined)).toBe(false);
		expect(isActive({ runId: 'r', view: null, error: null })).toBe(true);
		expect(isActive({ runId: 'r', view: view({ state: 'needs_approval' }), error: null })).toBe(true);
		expect(isActive({ runId: 'r', view: view({ state: 'done' }), error: null })).toBe(false);
	});
});

describe('toIdea', () => {
	const idea: BulleIdea = {
		id: 'i1', kind: 'task', title: 'Tidy', why: 'Why', draft: '', status: 'new', expires_at: '',
		sources: [{ label: 'Folder', ref: 'agenda:5/event:2' }, { label: 'Page', ref: 'https://example.com' }]
	};

	test('maps sources to links only when they are web addresses', () => {
		const card = toIdea(idea, undefined, t);
		expect(card.sources).toEqual([{ label: 'Folder', href: undefined }, { label: 'Page', href: 'https://example.com/' }]);
		expect(card.run).toBeUndefined();
	});

	test('links the run back to Bulle', () => {
		const card = toIdea(idea, { runId: 'r', view: view({ continue_url: 'https://bulle.facile.studio/chat/s1' }), error: null }, t);
		expect(card.continueHref).toBe('https://bulle.facile.studio/chat/s1');
	});
});

describe('errorKey and toolLabel', () => {
	test('maps statuses to messages', () => {
		expect(errorKey(new ApiError(429, 'x'))).toBe('ideas.error.capped');
		expect(errorKey(new ApiError(412, 'x'))).toBe('ideas.error.notLinked');
		expect(errorKey(new Error('x'))).toBe('ideas.error.unreachable');
	});

	test('unknown tools lose their prefix', () => {
		expect(toolLabel('agenda_move_event', t)).toBe('Move event');
	});
});
