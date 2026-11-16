import type { BullePage, Plan, PlanBlock } from '$lib/backend';

/** Bulle caps a run's prompt at 2000 characters. */
export const PROMPT_LIMIT = 2000;

/** The longest free-text ask the panel accepts, so the fixed template always fits. */
export const ASK_LIMIT = 500;

export type PlanKind = 'week' | 'today' | 'ask';

export type PlanRange = {
	/** First day to plan, YYYY-MM-DD in the user's zone. */
	from: string;
	/** Last day to plan, YYYY-MM-DD in the user's zone. */
	to: string;
	timezone: string;
	/** The moment the run starts: nothing is planned before it. */
	now: Date;
};

const RULES = [
	'Read the user\'s open tasks with opus_search_tasks and their free time with agenda_find_free_slots (pass the timezone).',
	'Save ONE draft with agenda_propose_plan. Never create, change or delete events yourself: the user applies the draft.',
	'Weigh each task\'s duration, priority and deadline. Split a task longer than 2 hours into sessions of 25 to 120 minutes.',
	'Keep dependent tasks in order. Leave 10 to 15 minutes around meetings.',
	'Stay inside work hours, 09:00 to 18:00 on weekdays, unless the user\'s request says otherwise.',
	'If a deadline cannot be met, say so in warnings instead of overbooking.',
	'Give every block a short reason, and put the task id in task_ref and its link in task_url.'
];

const SCOPE: Record<PlanKind, string> = {
	week: 'Plan the user\'s week',
	today: 'Plan the user\'s day',
	ask: 'Plan what the user asks for below'
};

/**
 * The prompt of a planning run: the app's fixed template, then the user's own request when they
 * typed one. Nothing from a calendar or a task goes in, so Bulle may read it as the owner's
 * message. The request is cut so the whole prompt stays under Bulle's cap.
 */
export function planPrompt(kind: PlanKind, range: PlanRange, ask = ''): string {
	const days = range.from === range.to ? range.from : `${range.from} to ${range.to}`;
	const head = [
		`${SCOPE[kind]}: ${days}, timezone ${range.timezone}. It is ${range.now.toISOString()} now.`,
		...RULES.map((rule, i) => `${i + 1}. ${rule}`)
	].join('\n');
	const request = ask.trim().slice(0, ASK_LIMIT);
	if (!request) return head;
	return `${head}\nThe user's request: ${request}`.slice(0, PROMPT_LIMIT);
}

/** The page context of a run: the visible calendar view, or nothing when the origin is not https. */
export function pageContext(view: string, label: string, origin: string): BullePage | undefined {
	if (!origin.startsWith('https://')) return undefined;
	return { app: 'agenda', id: `calendar-${view}`, title: label.slice(0, 120), url: `${origin}/calendar` };
}

/** The plan a run saved: the newest proposed plan created at or after the run started. */
export function planSince(plans: Plan[], startedAt: Date): Plan | null {
	const fresh = plans.filter((p) => p.state === 'proposed' && !p.expired && new Date(p.created_at) >= startedAt);
	fresh.sort((a, b) => b.created_at.localeCompare(a.created_at));
	return fresh[0] ?? null;
}

/** The blocks still to decide on: not yet applied. */
export function openBlocks(plan: Plan | null): PlanBlock[] {
	return plan?.state === 'proposed' ? plan.blocks.filter((b) => b.event_id === null) : [];
}

/** The open blocks that fall on a local day, earliest first. */
export function blocksOnDay(plan: Plan | null, day: Date): PlanBlock[] {
	return openBlocks(plan)
		.filter((b) => {
			const start = new Date(b.start_at);
			return start.getFullYear() === day.getFullYear() && start.getMonth() === day.getMonth() && start.getDate() === day.getDate();
		})
		.sort((a, b) => a.start_at.localeCompare(b.start_at));
}

/** The block ids a 409 from the apply route names: "... (block ids 12, 13). Apply again ...". */
export function conflictIds(message: string): number[] {
	const match = /block ids ([\d, ]+)/.exec(message);
	return match ? match[1]!.split(',').map((n) => Number(n.trim())).filter((n) => n > 0) : [];
}

/** YYYY-MM-DD of a local date. */
export function dayKey(date: Date): string {
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}
