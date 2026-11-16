<script lang="ts">
	import { BulleButton, type BulleMood } from '@facile/muse';
	import { bulleStore } from '$lib/bulle.svelte';
	import { t } from '$lib/i18n/index.svelte';

	const mood = $derived<BulleMood>(
		!bulleStore.available ? 'unavailable' : bulleStore.live ? 'thinking' : bulleStore.phase === 'done' ? 'done' : 'idle'
	);
</script>

{#if bulleStore.available}
	<BulleButton
		name={bulleStore.name}
		{mood}
		variant="icon"
		aria-expanded={bulleStore.panelOpen}
		aria-label={t('bulle.button')}
		onclick={() => (bulleStore.panelOpen = !bulleStore.panelOpen)}
	/>
{/if}
