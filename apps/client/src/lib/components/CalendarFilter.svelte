<script lang="ts">
	import { icons } from '@facile/muse';
	import type { CalendarItem } from '$lib/backend';
	import { calendarFilter, setCalendarFilter, type CalendarFilter } from '$lib/calendar-filter.svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { t } from '$lib/i18n/index.svelte';

	let { calendars, class: className = '' }: { calendars: CalendarItem[]; class?: string } = $props();

	const ALL = 'all';

	const active = $derived.by<CalendarFilter>(() => {
		const id = calendarFilter();
		return id !== ALL && calendars.some((c) => c.id === id) ? id : ALL;
	});
	const current = $derived(active === ALL ? null : (calendars.find((c) => c.id === active) ?? null));

	function choose(value: string) {
		setCalendarFilter(value === ALL ? ALL : Number(value));
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class="flex h-11 min-w-0 max-w-60 cursor-pointer items-center gap-2 rounded-fc-pill bg-muted px-3.5 text-sm font-medium transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring {className}"
		aria-label={t('calendar.filter')}
	>
		{#if current}
			<span class="size-2.5 shrink-0 rounded-full" style="background-color: {current.color}"></span>
		{:else}
			<iconify-icon icon="solar:layers-minimalistic-linear" width="16" height="16" class="block size-4"></iconify-icon>
		{/if}
		<span class="min-w-0 flex-1 truncate text-left">{current?.name ?? t('calendar.all')}</span>
		<iconify-icon icon={icons.chevronDown} width="16" height="16" class="block size-4 shrink-0 text-muted-foreground"></iconify-icon>
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="max-h-80 w-64">
		<DropdownMenu.RadioGroup value={String(active)} onValueChange={choose}>
			<DropdownMenu.RadioItem value={ALL} class="gap-2.5 py-2">
				<iconify-icon icon="solar:layers-minimalistic-linear" width="16" height="16" class="block size-4"></iconify-icon>
				{t('calendar.all')}
			</DropdownMenu.RadioItem>
			<DropdownMenu.Separator />
			{#each calendars as cal (cal.id)}
				<DropdownMenu.RadioItem value={String(cal.id)} class="gap-2.5 py-2" title={cal.space_id ? t('calendar.space', { name: cal.name }) : cal.name}>
					<span class="size-2.5 shrink-0 rounded-full" style="background-color: {cal.color}"></span>
					<span class="min-w-0 flex-1 truncate">{cal.name}</span>
					{#if cal.space_id}
						<iconify-icon icon={icons.space} width="14" height="14" class="block size-3.5 opacity-70"></iconify-icon>
					{/if}
				</DropdownMenu.RadioItem>
			{/each}
		</DropdownMenu.RadioGroup>
	</DropdownMenu.Content>
</DropdownMenu.Root>
