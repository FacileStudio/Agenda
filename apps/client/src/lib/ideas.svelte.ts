import { isNewIdea, type IdeaStatus } from '@facile/muse';
import { ApiError, backend, type CalendarItem, type IdeasResponse } from '$lib/backend';
import { calendarFilter } from '$lib/calendar-filter.svelte';
import type { AppKey } from '$lib/i18n/messages';
import { errorKey, isActive, type RunEntry } from '$lib/ideas';

const LIST_POLL_MS = 4000;
const RUN_POLL_MS = 2000;

/**
 * The active space's Agenda ideas, shared by the nav entry (badge and running dot) and the
 * Ideas page. Runs are polled here, so a run keeps going while the user browses elsewhere.
 */
class IdeasStore {
	page = $state<IdeasResponse | null>(null);
	loading = $state(false);
	error = $state<AppKey | null>(null);
	upgradeUrl = $state<string | null>(null);
	starting = $state(false);
	available = $state(true);
	runs = $state<Record<string, RunEntry>>({});
	busyIds = $state<string[]>([]);

	private spaceId: number | null = null;
	private listTimer: ReturnType<typeof setTimeout> | null = null;
	private runTimer: ReturnType<typeof setTimeout> | null = null;

	get generating(): boolean {
		return this.starting || Boolean(this.page?.generating);
	}

	get upsell(): boolean {
		return this.upgradeUrl !== null || this.page?.ai_access === false;
	}

	get running(): boolean {
		return this.generating || Object.values(this.runs).some(isActive);
	}

	get count(): number {
		return (this.page?.ideas ?? []).filter(
			(idea) =>
				!this.runs[idea.id] && isNewIdea({ status: idea.status as IdeaStatus, expiresAt: idea.expires_at || undefined })
		).length;
	}

	async load(spaceId: number | null): Promise<void> {
		if (spaceId !== this.spaceId) this.reset();
		this.spaceId = spaceId;
		this.loading = this.page === null;
		try {
			this.page = await backend.listIdeas(spaceId);
			this.error = null;
			this.available = true;
			this.upgradeUrl = this.page.ai_access ? null : (this.page.upgrade_url ?? '');
		} catch (error) {
			if (error instanceof ApiError && error.status === 404) this.available = false;
			else if (!this.refuse(error)) this.error = 'ideas.error.load';
		} finally {
			this.loading = false;
		}
		this.scheduleList();
	}

	async generate(): Promise<void> {
		this.starting = true;
		try {
			await backend.generateIdeas(this.spaceId);
			this.error = null;
			await this.load(this.spaceId);
		} catch (error) {
			if (!this.refuse(error)) this.error = errorKey(error);
		} finally {
			this.starting = false;
		}
	}

	async run(ideaId: string): Promise<void> {
		this.setRun(ideaId, { runId: null, view: null, error: null });
		try {
			const { run_id } = await backend.runIdea(ideaId, this.spaceId);
			this.setRun(ideaId, { runId: run_id, view: null, error: null });
			await this.poll();
		} catch (error) {
			if (this.refuse(error)) this.dropRun(ideaId);
			else this.setRun(ideaId, { runId: null, view: null, error: errorKey(error) });
		}
	}

	async decide(ideaId: string, callId: string, approve: boolean): Promise<void> {
		const runId = this.runs[ideaId]?.runId;
		if (!runId) return;
		await this.busy(ideaId, async () => {
			await backend.decideIdea(runId, callId, approve);
			await this.poll();
		});
	}

	async dismiss(ideaId: string): Promise<boolean> {
		return this.busy(ideaId, async () => {
			await backend.dismissIdea(ideaId);
			if (this.page) this.page = { ...this.page, ideas: this.page.ideas.filter((idea) => idea.id !== ideaId) };
			this.dropRun(ideaId);
		});
	}

	private reset(): void {
		if (this.listTimer) clearTimeout(this.listTimer);
		if (this.runTimer) clearTimeout(this.runTimer);
		this.listTimer = this.runTimer = null;
		this.page = null;
		this.runs = {};
		this.upgradeUrl = null;
		this.error = null;
	}

	private async busy(ideaId: string, work: () => Promise<void>): Promise<boolean> {
		this.busyIds = [...this.busyIds, ideaId];
		try {
			await work();
			return true;
		} catch (error) {
			if (!this.refuse(error)) this.error = errorKey(error);
			return false;
		} finally {
			this.busyIds = this.busyIds.filter((id) => id !== ideaId);
		}
	}

	private refuse(error: unknown): boolean {
		if (!(error instanceof ApiError) || error.status !== 402) return false;
		this.upgradeUrl = error.upgradeUrl ?? '';
		return true;
	}

	private setRun(ideaId: string, entry: RunEntry): void {
		this.runs = { ...this.runs, [ideaId]: entry };
	}

	private dropRun(ideaId: string): void {
		const { [ideaId]: _, ...rest } = this.runs;
		this.runs = rest;
	}

	private async poll(): Promise<void> {
		if (this.runTimer) clearTimeout(this.runTimer);
		this.runTimer = null;
		const active = Object.entries(this.runs).filter(([, entry]) => isActive(entry) && entry.runId);
		await Promise.all(
			active.map(async ([ideaId, entry]) => {
				try {
					const view = await backend.ideaRun(entry.runId!);
					this.setRun(ideaId, { ...entry, view });
				} catch (error) {
					this.setRun(ideaId, { ...entry, error: errorKey(error) });
				}
			})
		);
		if (Object.values(this.runs).some(isActive)) this.runTimer = setTimeout(() => void this.poll(), RUN_POLL_MS);
	}

	private scheduleList(): void {
		if (this.listTimer) clearTimeout(this.listTimer);
		this.listTimer = null;
		if (this.page?.generating) this.listTimer = setTimeout(() => void this.load(this.spaceId), LIST_POLL_MS);
	}
}

export const ideasStore = new IdeasStore();

/** The Agenda space the Ideas follow: the one of the calendar the views are narrowed to, none for personal or all. */
export function ideasSpaceId(calendars: CalendarItem[]): number | null {
	const id = calendarFilter();
	return id === 'all' ? null : (calendars.find((calendar) => calendar.id === id)?.space_id ?? null);
}
