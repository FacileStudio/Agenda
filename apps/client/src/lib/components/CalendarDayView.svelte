<script lang="ts">
	import { type CalendarDate, today, getLocalTimeZone } from '@internationalized/date';
	import { Badge } from '@facile/muse';
	import type { AgendaEvent, CalendarItem } from '$lib/backend';
	import { calendarColor, inkOn } from '$lib/calendar-colors';

	let {
		events,
		calendars,
		currentDate,
		onSlotClick,
		onEventClick
	}: {
		events: AgendaEvent[];
		calendars: CalendarItem[];
		currentDate: CalendarDate;
		onSlotClick: (hour: number) => void;
		onEventClick: (event: AgendaEvent) => void;
	} = $props();

	const tz = getLocalTimeZone();
	const HOURS = Array.from({ length: 24 }, (_, i) => i);
	const SLOT_HEIGHT_PX = 60;

	function getCalendarColor(calendarId: number): string {
		return calendarColor(calendars, calendarId);
	}

	function isSameDay(dateStr: string, cell: CalendarDate): boolean {
		const d = new Date(dateStr);
		return (
			d.getFullYear() === cell.year &&
			d.getMonth() + 1 === cell.month &&
			d.getDate() === cell.day
		);
	}

	function isMultiDay(event: AgendaEvent): boolean {
		const s = new Date(event.start_at);
		const e = new Date(event.end_at);
		return new Date(s.getFullYear(), s.getMonth(), s.getDate()).getTime() !==
			new Date(e.getFullYear(), e.getMonth(), e.getDate()).getTime();
	}

	function eventOverlapsDay(event: AgendaEvent, cell: CalendarDate): boolean {
		const start = new Date(event.start_at).getTime();
		const end = new Date(event.end_at).getTime();
		const d = cell.toDate(tz);
		const dayStart = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
		const dayEnd = dayStart + 86400000;
		return start < dayEnd && end > dayStart;
	}

	const todayDate = $derived(today(tz));

	const isToday = $derived(currentDate.compare(todayDate) === 0);

	const dayEvents = $derived(events.filter((e) => !e.is_all_day && !isMultiDay(e) && isSameDay(e.start_at, currentDate)));
	const allDayEvents = $derived(events.filter((e) => (e.is_all_day || isMultiDay(e)) && eventOverlapsDay(e, currentDate)));

	function getEventStyle(event: AgendaEvent): string {
		const start = new Date(event.start_at);
		const end = new Date(event.end_at);
		const startMinutes = start.getHours() * 60 + start.getMinutes();
		const endMinutes = end.getHours() * 60 + end.getMinutes();
		const duration = Math.max(endMinutes - startMinutes, 30);
		const top = (startMinutes / 60) * SLOT_HEIGHT_PX;
		const height = (duration / 60) * SLOT_HEIGHT_PX;
		return `top: ${top}px; height: ${height}px;`;
	}

	function formatHour(h: number): string {
		return h.toString().padStart(2, '0') + ':00';
	}
</script>

<div class="flex h-full flex-col overflow-hidden">
	<!-- All-day strip -->
	{#if allDayEvents.length > 0}
		<div class="flex items-center gap-1 border-b border-border px-4 py-2">
			<span class="w-12 shrink-0 text-xs text-muted-foreground">Jour</span>
			<div class="flex flex-wrap gap-1">
				{#each allDayEvents as event (event.id)}
					{@const fill = getCalendarColor(event.calendar_id)}
					<button
						type="button"
						class="cursor-pointer truncate rounded-md px-2 py-0.5 text-xs font-medium transition-[filter] hover:brightness-90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
						style="background-color: {fill}; color: {inkOn(fill)};"
						onclick={() => onEventClick(event)}
					>
						{event.title}
					</button>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Day column header -->
	<div class="flex items-center border-b border-border px-4 py-2">
		<div class="w-12"></div>
		<div class="flex flex-1 items-center justify-center gap-2 text-sm font-medium capitalize text-foreground">
			{currentDate.toDate(tz).toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' })}
			{#if isToday}
				<Badge tone="accent">Aujourd'hui</Badge>
			{/if}
		</div>
	</div>

	<!-- Time grid -->
	<div class="relative flex-1 overflow-y-auto">
		<div class="relative" style="height: {24 * SLOT_HEIGHT_PX}px;">
			<!-- Hour rows -->
			{#each HOURS as hour}
				<button
					type="button"
					aria-label={`Créer un événement à ${hour}h`}
					class="absolute right-0 left-0 cursor-pointer border-b border-dashed border-border transition-colors hover:bg-accent focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
					style="top: {hour * SLOT_HEIGHT_PX}px; height: {SLOT_HEIGHT_PX}px;"
					onclick={() => onSlotClick(hour)}
				>
					<span
						class="absolute left-0 top-0 w-12 pr-2 text-right text-xs text-muted-foreground"
						style={hour === 0 ? '' : 'transform: translateY(-50%);'}
					>
						{formatHour(hour)}
					</span>
				</button>
			{/each}

			<!-- Events -->
			{#each dayEvents as event (event.id)}
				{@const fill = getCalendarColor(event.calendar_id)}
				<button
					type="button"
					class="absolute right-2 left-14 flex cursor-pointer flex-col items-start overflow-hidden rounded-md px-1.5 py-1 text-left text-xs leading-tight font-medium shadow-sm transition-[filter] hover:brightness-90 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
					style="{getEventStyle(event)} background-color: {fill}; color: {inkOn(fill)};"
					onclick={() => onEventClick(event)}
				>
					<span class="block w-full truncate font-semibold">{event.title}</span>
					<span class="mt-0.5 block w-full truncate opacity-90">
						{new Date(event.start_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}
						–
						{new Date(event.end_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}
					</span>
					{#if event.location}
						<span class="block w-full truncate opacity-75">{event.location}</span>
					{/if}
				</button>
			{/each}
		</div>
	</div>
</div>
