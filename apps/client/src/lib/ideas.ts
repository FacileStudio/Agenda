import type { ApprovalRequest, Idea, IdeaRun } from '@facile/muse';
import type { AppKey } from '$lib/i18n/messages';
import { ApiError, type BulleIdea, type IdeaRunResponse } from '$lib/backend';

type Translate = (key: AppKey, params?: Record<string, string | number>) => string;

/** What the page knows about one idea's run: its id, the last poll, or why it stopped. */
export type RunEntry = { runId: string | null; view: IdeaRunResponse | null; error: AppKey | null };

const TOOL_KEYS: Record<string, AppKey> = {
	agenda_create_event: 'ideas.tool.agenda_create_event',
	agenda_update_event: 'ideas.tool.agenda_update_event',
	agenda_delete_event: 'ideas.tool.agenda_delete_event',
	agenda_read_event: 'ideas.tool.agenda_read_event',
	agenda_list_events: 'ideas.tool.agenda_list_events',
	agenda_list_calendars: 'ideas.tool.agenda_list_calendars',
	agenda_find_free_slots: 'ideas.tool.agenda_find_free_slots'
};

/** A readable name for a Bulle tool: Agenda's own are translated, others lose their prefix and underscores. */
export function toolLabel(tool: string, t: Translate): string {
	const key = TOOL_KEYS[tool];
	if (key) return t(key);
	const words = tool.replace(/^[a-z]+_/, '').replaceAll('_', ' ').trim();
	return words ? words[0]!.toUpperCase() + words.slice(1) : tool;
}

/** A link the page may render: an absolute http(s) URL, or nothing. */
export function safeHref(value: string | undefined): string | undefined {
	if (!value) return undefined;
	try {
		const url = new URL(value);
		return url.protocol === 'https:' || url.protocol === 'http:' ? url.toString() : undefined;
	} catch {
		return undefined;
	}
}

function approvalOf(view: IdeaRunResponse, t: Translate): ApprovalRequest | undefined {
	const first = view.pending_approvals?.[0];
	if (!first) return undefined;
	const tool = toolLabel(first.tool, t);
	const args = first.args && typeof first.args === 'object' ? (first.args as Record<string, unknown>) : undefined;
	return { id: first.call_id, tool, summary: t('ideas.approval.summary', { tool }), args };
}

/** The run muse draws for an entry: failed with a reason, running until the first poll, then Bulle's state. */
export function runOf(entry: RunEntry, t: Translate): IdeaRun {
	if (entry.error) return { state: 'failed', error: t(entry.error) };
	if (!entry.view) return { state: 'running' };
	const { view } = entry;
	if (view.state === 'failed') return { state: 'failed', error: view.summary || t('ideas.error.run') };
	const outputs = (view.outputs ?? []).flatMap((output) => {
		const href = safeHref(output.url);
		return href ? [{ label: output.preview || output.tool, href }] : [];
	});
	return { state: view.state, summary: view.summary || undefined, outputs, approval: approvalOf(view, t) };
}

/** True while the entry still needs polling. */
export function isActive(entry: RunEntry | undefined): boolean {
	if (!entry || entry.error) return false;
	return !entry.view || entry.view.state === 'running' || entry.view.state === 'needs_approval';
}

/** One card: Bulle's idea plus its run in this session, with the continue-in-Bulle link once it ran. */
export function toIdea(view: BulleIdea, entry: RunEntry | undefined, t: Translate): Idea {
	return {
		id: view.id,
		kind: view.kind,
		title: view.title,
		why: view.why,
		draft: view.draft || undefined,
		sources: (view.sources ?? []).map((source) => ({ label: source.label, href: safeHref(source.ref) })),
		status: view.status as Idea['status'],
		expiresAt: view.expires_at || undefined,
		run: entry ? runOf(entry, t) : undefined,
		continueHref: safeHref(entry?.view?.continue_url)
	};
}

const STATUS_KEYS: Record<number, AppKey> = {
	409: 'ideas.error.busy',
	412: 'ideas.error.notLinked',
	404: 'ideas.error.gone',
	429: 'ideas.error.capped'
};

/** The message key of a failed call. */
export function errorKey(error: unknown): AppKey {
	return (error instanceof ApiError && STATUS_KEYS[error.status]) || 'ideas.error.unreachable';
}
