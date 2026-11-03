<script lang="ts">
	import { icons } from '@facile/muse';
	import type { CalendarItem } from '$lib/backend';
	import { calendarFilter, setCalendarFilter, type CalendarFilter } from '$lib/calendar-filter.svelte';

	let { calendars, class: className = '' }: { calendars: CalendarItem[]; class?: string } = $props();

	const active = $derived.by<CalendarFilter>(() => {
		const id = calendarFilter();
		return id !== 'all' && calendars.some((c) => c.id === id) ? id : 'all';
	});

	const chip =
		'flex h-9 shrink-0 cursor-pointer items-center gap-2 rounded-fc-pill px-3.5 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring';

	function tone(selected: boolean) {
		return selected ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground';
	}
</script>

<div
	role="toolbar"
	aria-label="Filtrer les calendriers"
	class="flex min-w-0 items-center gap-1 overflow-x-auto rounded-fc-pill bg-muted p-1 [scrollbar-width:none] {className}"
>
	<button
		type="button"
		class="{chip} {tone(active === 'all')}"
		aria-pressed={active === 'all'}
		onclick={() => setCalendarFilter('all')}
	>
		<iconify-icon icon="solar:layers-minimalistic-linear" width="16" height="16" class="block size-4"></iconify-icon>
		Tous
	</button>
	{#each calendars as cal (cal.id)}
		{@const selected = active === cal.id}
		<button
			type="button"
			class="{chip} {tone(selected)}"
			aria-pressed={selected}
			title={cal.space_id ? `Espace ${cal.name}` : cal.name}
			onclick={() => setCalendarFilter(cal.id)}
		>
			<span
				class="size-2.5 shrink-0 rounded-full ring-2 {selected ? 'ring-primary-foreground/40' : 'ring-transparent'}"
				style="background-color: {cal.color}"
			></span>
			<span class="max-w-40 truncate">{cal.name}</span>
			{#if cal.space_id}
				<iconify-icon icon={icons.space} width="14" height="14" class="block size-3.5 opacity-70"></iconify-icon>
			{/if}
		</button>
	{/each}
</div>
