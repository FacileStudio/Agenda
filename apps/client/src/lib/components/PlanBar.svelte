<script lang="ts">
	import { Button, icons } from '@facile/muse';
	import { bulleStore } from '$lib/bulle.svelte';
	import { openBlocks } from '$lib/bulle';
	import { t } from '$lib/i18n/index.svelte';

	let { onApplied }: { onApplied: () => Promise<void> } = $props();

	const count = $derived(openBlocks(bulleStore.plan).length);
</script>

{#if bulleStore.plan && count > 0}
	<div class="mx-3 mb-2 flex flex-wrap items-center gap-2 rounded-fc-md bg-fc-component px-3 py-2 sm:mx-5" role="region" aria-label={t('bulle.plan.heading')}>
		<p class="m-0 min-w-0 flex-1 basis-48 text-fc-sm text-fc-fg">{t('bulle.plan.bar', { name: bulleStore.name, count })}</p>
		<div class="flex flex-wrap items-center gap-2">
			<Button size="lg" icon={icons.check} disabled={bulleStore.applying || bulleStore.kept.length === 0} onclick={() => void bulleStore.apply(false, onApplied)}>
				{bulleStore.kept.length === count ? t('bulle.plan.acceptAll') : t('bulle.plan.accept', { count: bulleStore.kept.length })}
			</Button>
			<Button size="lg" variant="secondary" onclick={() => (bulleStore.panelOpen = true)}>{t('bulle.plan.review')}</Button>
			<Button size="lg" variant="ghost" disabled={bulleStore.applying} onclick={() => void bulleStore.dismiss()}>{t('bulle.plan.dismiss')}</Button>
		</div>
	</div>
{/if}
