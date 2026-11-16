import { toast } from '@facile/muse';
import { ApiError, backend, type BullePage, type IdeaRunResponse, type Plan, type PlanBlock } from '$lib/backend';
import { conflictIds, openBlocks, planSince, type PlanKind, type PlanRange, planPrompt } from '$lib/bulle';
import type { AppKey } from '$lib/i18n/messages';
import { isActive, type RunEntry } from '$lib/ideas';
import { t } from '$lib/i18n/index.svelte';

const RUN_POLL_MS = 2000;
const CLOCK_SLACK_MS = 2 * 60 * 1000;

export type RunPhase = 'idle' | 'running' | 'done' | 'noPlan' | 'failed';

/**
 * The calendar's Bulle button: the assistant's profile, one planning run at a time, and the draft
 * plan it saved. The run is polled here, so it keeps going while the panel is closed. The plan is
 * read from /api/plans, never from the run's text: Bulle only proposes, the user applies.
 */
class BulleStore {
	name = $state('Bulle');
	available = $state(true);
	panelOpen = $state(false);
	upgradeUrl = $state<string | null>(null);
	run = $state<RunEntry | null>(null);
	phase = $state<RunPhase>('idle');
	starting = $state(false);
	approvalBusy = $state(false);
	error = $state<AppKey | null>(null);
	plan = $state<Plan | null>(null);
	dropped = $state<number[]>([]);
	selected = $state<number | null>(null);
	conflict = $state<number[] | null>(null);
	applying = $state(false);

	private startedAt = new Date();
	private timer: ReturnType<typeof setTimeout> | null = null;
	private spaceId: number | null = null;

	get live(): boolean {
		return this.starting || (this.run !== null && isActive(this.run));
	}

	get kept(): PlanBlock[] {
		return openBlocks(this.plan).filter((b) => !this.dropped.includes(b.id));
	}

	async load(spaceId: number | null): Promise<void> {
		this.spaceId = spaceId;
		try {
			const profile = await backend.bulleProfile();
			this.name = profile.name?.trim() || 'Bulle';
			this.available = true;
		} catch (error) {
			this.available = !(error instanceof ApiError && (error.status === 404 || error.status >= 500));
		}
		await this.loadPlan(null);
	}

	async loadPlan(since: Date | null): Promise<void> {
		try {
			const plans = await backend.listPlans('proposed');
			const next = since ? planSince(plans, since) : (plans.find((p) => !p.expired) ?? null);
			this.setPlan(next);
		} catch {
			this.error = 'bulle.error.plan';
		}
	}

	async start(kind: PlanKind, range: PlanRange, ask: string, page: BullePage | undefined): Promise<void> {
		if (this.live) return;
		this.starting = true;
		this.error = null;
		this.phase = 'running';
		this.run = { runId: null, view: null, error: null };
		this.startedAt = new Date(Date.now() - CLOCK_SLACK_MS);
		try {
			const title = kind === 'week' ? t('bulle.action.week') : kind === 'today' ? t('bulle.action.today') : t('bulle.title', { name: this.name });
			const { run_id } = await backend.startBulleRun(planPrompt(kind, range, ask), title, page, this.spaceId);
			this.run = { runId: run_id, view: null, error: null };
			await this.poll();
		} catch (error) {
			this.run = null;
			this.phase = 'idle';
			if (!this.refuse(error)) this.error = error instanceof ApiError && error.status === 409 ? 'ideas.error.busy' : 'bulle.error.start';
		} finally {
			this.starting = false;
		}
	}

	async decide(callId: string, approve: boolean): Promise<void> {
		const runId = this.run?.runId;
		if (!runId) return;
		this.approvalBusy = true;
		try {
			await backend.decideBulleRun(runId, callId, approve);
			await this.poll();
		} catch (error) {
			if (!this.refuse(error)) this.error = 'bulle.error.start';
		} finally {
			this.approvalBusy = false;
		}
	}

	toggle(id: number): void {
		this.dropped = this.dropped.includes(id) ? this.dropped.filter((d) => d !== id) : [...this.dropped, id];
		this.conflict = null;
	}

	leaveOut(ids: number[]): void {
		this.dropped = [...new Set([...this.dropped, ...ids])];
		this.conflict = null;
	}

	select(id: number | null): void {
		this.selected = id;
	}

	async apply(force: boolean, onApplied: () => Promise<void>): Promise<void> {
		const plan = this.plan;
		const ids = this.kept.map((b) => b.id);
		if (!plan || this.applying || ids.length === 0) return;
		this.applying = true;
		this.error = null;
		try {
			const result = await backend.applyPlan(plan.id, ids, force);
			toast.success(t('bulle.plan.applied', { count: result.events.length }));
			this.setPlan(null);
			this.conflict = null;
			await onApplied();
		} catch (error) {
			this.failApply(error);
		} finally {
			this.applying = false;
		}
	}

	async dismiss(): Promise<void> {
		const plan = this.plan;
		if (!plan || this.applying) return;
		this.applying = true;
		try {
			await backend.dismissPlan(plan.id);
			this.setPlan(null);
			toast.info(t('bulle.plan.dismissed'));
		} catch (error) {
			if (error instanceof ApiError && error.status === 412) this.setPlan(null);
			else this.error = 'bulle.error.apply';
		} finally {
			this.applying = false;
		}
	}

	private failApply(error: unknown): void {
		if (error instanceof ApiError && error.status === 409) {
			this.conflict = conflictIds(error.message);
			this.panelOpen = true;
			return;
		}
		if (error instanceof ApiError && error.status === 412) {
			this.error = 'bulle.error.expired';
			this.setPlan(null);
			return;
		}
		this.error = 'bulle.error.apply';
		this.panelOpen = true;
	}

	private setPlan(plan: Plan | null): void {
		this.plan = plan;
		this.dropped = [];
		this.selected = null;
		this.conflict = null;
	}

	private refuse(error: unknown): boolean {
		if (!(error instanceof ApiError) || error.status !== 402) return false;
		this.upgradeUrl = error.upgradeUrl ?? '';
		return true;
	}

	private async poll(): Promise<void> {
		if (this.timer) clearTimeout(this.timer);
		this.timer = null;
		const entry = this.run;
		if (!entry?.runId) return;
		try {
			const view: IdeaRunResponse = await backend.bulleRun(entry.runId);
			this.run = { ...entry, view };
		} catch {
			this.run = { ...entry, error: 'bulle.error.start' };
		}
		if (this.run && isActive(this.run)) {
			this.timer = setTimeout(() => void this.poll(), RUN_POLL_MS);
			return;
		}
		await this.finish();
	}

	private async finish(): Promise<void> {
		const view = this.run?.view;
		if (!view || view.state === 'failed' || this.run?.error) {
			this.phase = 'failed';
			return;
		}
		const since = view.started_at ? new Date(view.started_at) : this.startedAt;
		await this.loadPlan(since);
		this.phase = this.plan ? 'done' : 'noPlan';
	}
}

export const bulleStore = new BulleStore();
