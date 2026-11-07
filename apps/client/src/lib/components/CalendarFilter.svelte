<script lang="ts">
	import { Icon, Menu, icons } from '@facile/muse';
	import type { CalendarItem } from '$lib/backend';
	import { calendarFilter, setCalendarFilter, type CalendarFilter } from '$lib/calendar-filter.svelte';
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

<Menu
	label={t('calendar.filter')}
	class={className}
	items={[
		{
			label: t('calendar.all'),
			icon: active === ALL ? icons.check : 'solar:layers-minimalistic-linear',
			onselect: () => choose(ALL)
		},
		{ separator: true },
		...calendars.map((cal) => ({
			label: cal.name,
			icon: active === cal.id ? icons.check : cal.space_id ? icons.space : icons.calendar,
			onselect: () => choose(String(cal.id))
		}))
	]}
>
	{#snippet trigger()}
		{#if current}
			<span class="size-2.5 shrink-0 rounded-full" style="background-color: {current.color}"></span>
		{:else}
			<Icon icon="solar:layers-minimalistic-linear" size={16} class="shrink-0" />
		{/if}
		<span class="max-w-48 min-w-0 truncate text-left font-medium">{current?.name ?? t('calendar.all')}</span>
		<Icon icon={icons.chevronDown} size={16} class="shrink-0 text-fc-fg-muted" />
	{/snippet}
</Menu>
