<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { IdeasPage, IdeasUpsell, toast, type ApprovalRequest, type Idea } from '@facile/muse';
	import { REGISTRE_SPACES_URL, type CalendarItem } from '$lib/backend';
	import { ideasSpaceId, ideasStore } from '$lib/ideas.svelte';
	import { toIdea } from '$lib/ideas';
	import { t } from '$lib/i18n/index.svelte';

	const app = getContext<{ calendars: CalendarItem[] }>('app');

	const cards = $derived((ideasStore.page?.ideas ?? []).map((idea) => toIdea(idea, ideasStore.runs[idea.id], t)));
	const upgradeHref = $derived(safeUpgrade(ideasStore.upgradeUrl || ideasStore.page?.upgrade_url));

	const examples = $derived([
		{ kind: 'event', title: t('ideas.example.prep.title'), why: t('ideas.example.prep.why') },
		{ kind: 'task', title: t('ideas.example.clash.title'), why: t('ideas.example.clash.why') },
		{ kind: 'follow_up', title: t('ideas.example.followup.title'), why: t('ideas.example.followup.why') }
	]);

	function safeUpgrade(value: string | null | undefined): string {
		try {
			const url = new URL(value ?? '');
			return url.protocol === 'https:' ? url.toString() : REGISTRE_SPACES_URL;
		} catch {
			return REGISTRE_SPACES_URL;
		}
	}

	onMount(() => void ideasStore.load(ideasSpaceId(app.calendars)));

	async function dismiss(idea: Idea) {
		if (await ideasStore.dismiss(idea.id)) toast.info(t('ideas.dismissed'));
	}

	function answer(idea: Idea, approval: ApprovalRequest, approve: boolean) {
		void ideasStore.decide(idea.id, approval.id, approve);
	}
</script>

<svelte:head>
	<title>{t('ideas.title')}</title>
</svelte:head>

<div class="mx-auto w-full max-w-5xl px-4 py-6 md:px-8">
	{#if ideasStore.upsell}
		<IdeasUpsell {upgradeHref} {examples} />
	{:else}
		<IdeasPage
			ideas={cards}
			loading={ideasStore.loading}
			generating={ideasStore.generating}
			error={ideasStore.error ? t(ideasStore.error) : null}
			lastGeneratedAt={ideasStore.page?.last_batch_at ?? null}
			busyIds={ideasStore.busyIds}
			description={t('ideas.description')}
			onGenerate={() => void ideasStore.generate()}
			onRetry={() => void ideasStore.load(ideasSpaceId(app.calendars))}
			onRun={(idea) => void ideasStore.run(idea.id)}
			onDismiss={dismiss}
			onApprove={(idea, approval) => answer(idea, approval, true)}
			onDecline={(idea, approval) => answer(idea, approval, false)}
		/>
	{/if}
</div>
