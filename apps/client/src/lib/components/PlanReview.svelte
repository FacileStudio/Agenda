<script lang="ts">
	import { Alert, Badge, Button, icons } from '@facile/muse';
	import { bulleStore } from '$lib/bulle.svelte';
	import { openBlocks } from '$lib/bulle';
	import { formatDate, formatTime } from '$lib/i18n/format';
	import { t } from '$lib/i18n/index.svelte';

	let { onApplied }: { onApplied: () => Promise<void> } = $props();

	const store = bulleStore;
	const blocks = $derived(openBlocks(store.plan));
	const everything = $derived(store.kept.length === blocks.length);
	const clashing = $derived(blocks.filter((b) => store.conflict?.includes(b.id)));
	const clashText = $derived(
		clashing.length ? t('bulle.plan.conflict', { titles: clashing.map((b) => b.title).join(', ') }) : t('bulle.plan.conflictGeneric')
	);

	function when(start: string, end: string): string {
		return `${formatDate(start, { weekday: 'short', day: 'numeric', month: 'short' })}, ${formatTime(start)} – ${formatTime(end)}`;
	}
</script>

{#if store.plan}
	<section class="flex flex-col gap-3" aria-label={t('bulle.plan.heading')}>
		<div class="flex flex-wrap items-center justify-between gap-2">
			<h3 class="m-0 text-fc-base font-medium text-fc-fg">{store.plan.title}</h3>
			<Badge tone="neutral">{t('bulle.plan.summary', { count: blocks.length })}</Badge>
		</div>

		{#if store.plan.warnings.length}
			<Alert tone="warning" class="flex flex-col gap-1">
				<span class="font-medium">{t('bulle.plan.warnings')}</span>
				<ul class="m-0 list-disc pl-5">
					{#each store.plan.warnings as warning}
						<li class="wrap-anywhere">{warning}</li>
					{/each}
				</ul>
			</Alert>
		{/if}

		{#if store.conflict}
			<Alert tone="warning" class="flex flex-col items-start gap-3" role="alert">
				<span class="wrap-anywhere">{clashText}</span>
				<div class="flex flex-wrap gap-2">
					<Button size="lg" variant="secondary" disabled={store.applying} onclick={() => void store.apply(true, onApplied)}>
						{t('bulle.plan.force')}
					</Button>
					{#if store.conflict.length}
						<Button size="lg" variant="ghost" disabled={store.applying} onclick={() => store.leaveOut(store.conflict ?? [])}>
							{t('bulle.plan.leaveOut')}
						</Button>
					{/if}
				</div>
			</Alert>
		{/if}

		<ul class="m-0 flex list-none flex-col gap-2 p-0">
			{#each blocks as block (block.id)}
				{@const dropped = store.dropped.includes(block.id)}
				<li class="rounded-fc-md px-3 py-2 {store.selected === block.id ? 'bg-fc-component' : 'bg-fc-surface'}">
					<div class="flex items-center gap-2">
						<button
							type="button"
							class="min-h-11 min-w-0 flex-1 cursor-pointer text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
							onclick={() => store.select(store.selected === block.id ? null : block.id)}
							aria-expanded={store.selected === block.id}
						>
							<span class="block truncate text-fc-sm font-medium text-fc-fg {dropped ? 'line-through opacity-60' : ''}">{block.title}</span>
							<span class="block truncate text-fc-xs text-fc-fg-muted">{when(block.start_at, block.end_at)}</span>
						</button>
						<Button
							size="lg"
							variant={dropped ? 'secondary' : 'ghost'}
							icon={dropped ? icons.plus : icons.close}
							aria-pressed={!dropped}
							onclick={() => store.toggle(block.id)}
						>
							{dropped ? t('bulle.plan.keep') : t('bulle.plan.drop')}
						</Button>
					</div>
					{#if store.conflict?.includes(block.id)}
						<p class="m-0 text-fc-xs text-fc-warning">{t('bulle.plan.overlap')}</p>
					{/if}
					{#if block.reason && (store.selected === block.id || blocks.length <= 3)}
						<p class="mt-1 mb-0 text-fc-xs wrap-anywhere text-fc-fg-muted">{t('bulle.plan.reason', { reason: block.reason })}</p>
					{/if}
				</li>
			{/each}
		</ul>

		<div class="flex flex-wrap items-center gap-2">
			<Button size="lg" icon={icons.check} disabled={store.applying || store.kept.length === 0} onclick={() => void store.apply(false, onApplied)}>
				{everything ? t('bulle.plan.acceptAll') : t('bulle.plan.accept', { count: store.kept.length })}
			</Button>
			<Button size="lg" variant="ghost" disabled={store.applying} onclick={() => void store.dismiss()}>
				{t('bulle.plan.dismiss')}
			</Button>
		</div>
	</section>
{/if}
