<script lang="ts">
	import { ApprovalCard, BullePanel, Button, Drawer, IconButton, Textarea, icons, type ApprovalRequest } from '@facile/muse';
	import { REGISTRE_SPACES_URL } from '$lib/backend';
	import { ASK_LIMIT, dayKey, pageContext, type PlanKind } from '$lib/bulle';
	import { bulleStore } from '$lib/bulle.svelte';
	import PlanReview from '$lib/components/PlanReview.svelte';
	import { ideasStore } from '$lib/ideas.svelte';
	import { runOf, safeHref } from '$lib/ideas';
	import { t } from '$lib/i18n/index.svelte';

	let {
		view,
		visibleDate,
		weekStart,
		weekEnd,
		periodLabel,
		sidePanel,
		onApplied
	}: {
		view: string;
		visibleDate: Date;
		weekStart: Date;
		weekEnd: Date;
		periodLabel: string;
		sidePanel: boolean;
		onApplied: () => Promise<void>;
	} = $props();

	const store = bulleStore;
	let ask = $state('');

	const upsell = $derived(store.upgradeUrl !== null || ideasStore.upsell);
	const upgradeHref = $derived(upsell ? (safeHref(store.upgradeUrl || ideasStore.upgradeUrl || '') ?? REGISTRE_SPACES_URL) : undefined);
	const runView = $derived(store.run ? runOf(store.run, t) : null);
	const approval = $derived<ApprovalRequest | undefined>(runView?.approval);
	const error = $derived(store.error ? [t(store.error), store.errorDetail].filter(Boolean).join(' ') : null);
	const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

	function start(kind: PlanKind) {
		const now = new Date();
		const today = dayKey(now);
		const first = kind === 'week' ? (weekStart > now ? dayKey(weekStart) : today) : kind === 'today' ? today : dayKey(visibleDate > now ? visibleDate : now);
		const last = kind === 'week' ? dayKey(weekEnd) : first;
		const page = pageContext(view, periodLabel, location.origin);
		void store.start(kind, { from: first, to: last, timezone, now }, kind === 'ask' ? ask : '', page).then(() => {
			if (kind === 'ask' && !store.error) ask = '';
		});
	}

	function close() {
		store.panelOpen = false;
	}

</script>

{#snippet closeButton()}
	<IconButton variant="ghost" aria-label={t('bulle.close')} onclick={close}>
		<iconify-icon icon={icons.close} width="20" height="20" class="block size-5"></iconify-icon>
	</IconButton>
{/snippet}

{#snippet body()}
	<BullePanel
		name={store.name}
		title={t('bulle.title', { name: store.name })}
		busy={store.live}
		error={error}
		onRetry={store.error === 'bulle.error.plan' ? () => void store.loadPlan(null) : undefined}
		{upgradeHref}
		upsellText={t('bulle.upsell', { name: store.name })}
		actions={sidePanel ? closeButton : undefined}
		class={sidePanel ? '' : 'bg-transparent p-0'}
	>
		<div class="flex flex-col gap-4">
			<p class="m-0 text-fc-sm text-fc-fg-muted">{t('bulle.intro', { name: store.name })}</p>

			<div class="flex flex-wrap gap-2">
				<Button size="lg" icon={icons.calendar} disabled={store.live} onclick={() => start('week')}>{t('bulle.action.week')}</Button>
				<Button size="lg" variant="secondary" icon={icons.calendar} disabled={store.live} onclick={() => start('today')}>{t('bulle.action.today')}</Button>
			</div>

			<form
				class="flex flex-col gap-2"
				onsubmit={(e) => {
					e.preventDefault();
					if (ask.trim()) start('ask');
				}}
			>
				<label class="text-fc-sm font-medium text-fc-fg" for="bulle-ask">{t('bulle.ask.label')}</label>
				<Textarea id="bulle-ask" bind:value={ask} rows={3} maxlength={ASK_LIMIT} placeholder={t('bulle.ask.placeholder')} disabled={store.live} />
				<Button type="submit" size="lg" class="self-start" disabled={store.live || !ask.trim()}>{t('bulle.ask.send')}</Button>
			</form>

			{#if approval}
				<div class="flex flex-col gap-2">
					<h3 class="m-0 text-fc-sm font-medium text-fc-fg">{t('bulle.approvals')}</h3>
					<ApprovalCard
						{approval}
						busy={store.approvalBusy}
						onApprove={(a) => void store.decide(a.id, true)}
						onDecline={(a) => void store.decide(a.id, false)}
					/>
				</div>
			{/if}

			{#if !store.live}
				{#if store.phase === 'failed'}
					<p class="m-0 text-fc-sm text-fc-danger" role="status">{runView?.error || t('bulle.run.failed')}</p>
				{:else if store.phase === 'noPlan'}
					<p class="m-0 text-fc-sm text-fc-fg-muted" role="status">{t('bulle.run.noPlan', { name: store.name })}</p>
				{:else if store.phase === 'done'}
					<p class="m-0 text-fc-sm text-fc-fg-muted" role="status">{t('bulle.run.done')}</p>
				{/if}
				{#if safeHref(store.run?.view?.continue_url)}
					<a class="text-fc-sm text-fc-accent underline" href={safeHref(store.run?.view?.continue_url)}>{t('bulle.run.continue')}</a>
				{/if}
			{/if}

			<PlanReview {onApplied} />
		</div>
	</BullePanel>
{/snippet}

{#if store.available}
	{#if sidePanel}
		{#if store.panelOpen}
			<aside class="w-[26rem] shrink-0 overflow-y-auto pr-5 pb-5" aria-label={t('bulle.title', { name: store.name })}>
				{@render body()}
			</aside>
		{/if}
	{:else}
		<Drawer bind:open={store.panelOpen} showClose>
			<div class="max-h-[70dvh] overflow-y-auto px-4 pb-4">
				{@render body()}
			</div>
		</Drawer>
	{/if}
{/if}
