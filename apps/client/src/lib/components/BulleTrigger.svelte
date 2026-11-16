<script lang="ts">
	import { BulleButton, type BulleMood } from '@facile/muse';
	import { bulleStore } from '$lib/bulle.svelte';
	import { t } from '$lib/i18n/index.svelte';

	let { wide = false }: { wide?: boolean } = $props();

	const mood = $derived<BulleMood>(
		!bulleStore.available ? 'unavailable' : bulleStore.live ? 'thinking' : bulleStore.phase === 'done' ? 'done' : 'idle'
	);
</script>

{#if bulleStore.available}
	<BulleButton
		name={bulleStore.name}
		{mood}
		variant={wide ? 'header' : 'icon'}
		label={wide ? bulleStore.name : undefined}
		aria-expanded={bulleStore.panelOpen}
		aria-label={t('bulle.title', { name: bulleStore.name })}
		onclick={() => (bulleStore.panelOpen = !bulleStore.panelOpen)}
	/>
{/if}
