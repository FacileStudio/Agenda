<script lang="ts">
	import { Badge } from '@facile/muse';
	import type { AgendaEvent } from '$lib/backend';

	let { event, class: className = '' }: { event: AgendaEvent; class?: string } = $props();
</script>

{#if event.source === 'opus'}
	<Badge class="ml-auto shrink-0 px-1.5 py-0 text-[0.625rem] {className}">Opus</Badge>
{/if}
