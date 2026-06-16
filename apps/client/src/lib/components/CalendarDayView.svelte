<script lang="ts">
	import { type CalendarDate, today, getLocalTimeZone } from '@internationalized/date';
	import type { AgendaEvent, CalendarItem } from '$lib/backend';

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
	const SLOT_HEIGHT = 60; // px per hour

	function getCalendarColor(calendarId: number): string {
		return calendars.find((c) => c.id === calendarId)?.color ?? '#6b7280';
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
		const top = (startMinutes / 60) * SLOT_HEIGHT;
		const height = (duration / 60) * SLOT_HEIGHT;
		return `top: ${top}px; height: ${height}px;`;
	}

	function formatHour(h: number): string {
		return h.toString().padStart(2, '0') + ':00';
	}
</script>

<div class="flex h-full flex-col overflow-hidden">
	<!-- All-day strip -->
	{#if allDayEvents.length > 0}
		<div class="flex items-center gap-1 border-b px-4 py-2">
			<span class="w-12 flex-shrink-0 text-xs text-muted-foreground">Jour</span>
			<div class="flex flex-wrap gap-1">
				{#each allDayEvents as event (event.id)}
					<button
						class="cursor-pointer truncate rounded px-2 py-0.5 text-xs font-medium text-white transition-[filter] hover:brightness-90"
						style="background-color: {getCalendarColor(event.calendar_id)}"
						onclick={() => onEventClick(event)}
					>
						{event.title}
					</button>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Day column header -->
	<div class="flex items-center border-b px-4 py-2">
		<div class="w-12"></div>
		<div
			class="flex flex-1 items-center justify-center gap-2 text-sm font-medium
				{isToday ? 'text-primary' : 'text-foreground'}"
		>
			{currentDate.toDate(tz).toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' })}
		</div>
	</div>

	<!-- Time grid -->
	<div class="relative flex-1 overflow-y-auto">
		<div class="relative" style="height: {24 * SLOT_HEIGHT}px;">
			<!-- Hour rows -->
			{#each HOURS as hour}
				<button
					class="absolute left-0 right-0 cursor-pointer border-b border-dashed border-border/50 hover:bg-accent/30"
					style="top: {hour * SLOT_HEIGHT}px; height: {SLOT_HEIGHT}px;"
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
				<button
					class="absolute left-14 right-2 cursor-pointer overflow-hidden rounded px-1.5 py-0.5 text-left text-xs font-medium text-white shadow-sm transition-[filter] hover:brightness-90"
					style="{getEventStyle(event)} background-color: {getCalendarColor(event.calendar_id)};"
					onclick={() => onEventClick(event)}
				>
					<span class="block truncate font-semibold">{event.title}</span>
					<span class="block truncate opacity-90">
						{new Date(event.start_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}
						–
						{new Date(event.end_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}
					</span>
					{#if event.location}
						<span class="block truncate opacity-75">{event.location}</span>
					{/if}
				</button>
			{/each}
		</div>
	</div>
</div>
