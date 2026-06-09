<script lang="ts">
	import {
		type CalendarDate,
		today,
		getLocalTimeZone,
		startOfWeek,
		endOfWeek
	} from '@internationalized/date';
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
		onSlotClick: (date: CalendarDate, hour: number) => void;
		onEventClick: (event: AgendaEvent) => void;
	} = $props();

	const tz = getLocalTimeZone();
	const HOURS = Array.from({ length: 24 }, (_, i) => i);
	const SLOT_HEIGHT = 60; // px per hour

	function getCalendarColor(calendarId: number): string {
		return calendars.find((c) => c.id === calendarId)?.color ?? '#6b7280';
	}

	const todayDate = $derived(today(tz));

	const weekDays = $derived(() => {
		const weekStart = startOfWeek(currentDate, 'fr-FR');
		return Array.from({ length: 7 }, (_, i) => weekStart.add({ days: i }));
	});

	function isSameDay(dateStr: string, cell: CalendarDate): boolean {
		const d = new Date(dateStr);
		return (
			d.getFullYear() === cell.year &&
			d.getMonth() + 1 === cell.month &&
			d.getDate() === cell.day
		);
	}

	function eventsForDay(cell: CalendarDate): AgendaEvent[] {
		return events.filter((e) => !e.is_all_day && isSameDay(e.start_at, cell));
	}

	function allDayEventsForDay(cell: CalendarDate): AgendaEvent[] {
		return events.filter((e) => e.is_all_day && isSameDay(e.start_at, cell));
	}

	function isToday(cell: CalendarDate): boolean {
		return cell.compare(todayDate) === 0;
	}

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

	const DAY_ABBREVS = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim'];
</script>

<div class="flex h-full flex-col overflow-hidden">
	<!-- All-day strip -->
	<div class="flex border-b">
		<div class="w-12 flex-shrink-0 border-r py-1"></div>
		{#each weekDays() as day, di (day.toString())}
			<div class="flex flex-1 flex-col border-r last:border-r-0">
				<!-- Day header -->
				<div
					class="flex items-center justify-center gap-1 py-1 text-xs
						{isToday(day) ? 'font-bold text-primary' : 'text-muted-foreground'}"
				>
					{DAY_ABBREVS[di]}
					<span
						class="flex size-5 items-center justify-center rounded-full text-xs
							{isToday(day) ? 'bg-primary text-primary-foreground font-bold' : ''}"
					>
						{day.day}
					</span>
				</div>
				<!-- All-day events -->
				<div class="min-h-6 px-0.5 pb-1">
					{#each allDayEventsForDay(day) as event (event.id)}
						<button
							class="mb-0.5 w-full truncate rounded px-1 py-0.5 text-left text-xs font-medium text-white"
							style="background-color: {getCalendarColor(event.calendar_id)}"
							onclick={() => onEventClick(event)}
						>
							{event.title}
						</button>
					{/each}
				</div>
			</div>
		{/each}
	</div>

	<!-- Time grid -->
	<div class="flex flex-1 overflow-y-auto">
		<!-- Time labels -->
		<div class="w-12 flex-shrink-0 border-r">
			<div class="relative" style="height: {24 * SLOT_HEIGHT}px;">
				{#each HOURS as hour}
					<div
						class="absolute left-0 right-0 pr-1 text-right text-xs text-muted-foreground"
						style="top: {hour * SLOT_HEIGHT}px;"
					>
						{formatHour(hour)}
					</div>
				{/each}
			</div>
		</div>

		<!-- Day columns -->
		{#each weekDays() as day, di (day.toString())}
			<div class="relative flex-1 border-r last:border-r-0">
				<div class="relative" style="height: {24 * SLOT_HEIGHT}px;">
					<!-- Hour slot backgrounds -->
					{#each HOURS as hour}
						<button
							aria-label={`Créer un événement à ${hour}h`}
							class="absolute left-0 right-0 border-b border-dashed border-border/50 hover:bg-accent/30"
							style="top: {hour * SLOT_HEIGHT}px; height: {SLOT_HEIGHT}px;"
							onclick={() => onSlotClick(day, hour)}
						></button>
					{/each}

					<!-- Events -->
					{#each eventsForDay(day) as event (event.id)}
						<button
							class="absolute left-0.5 right-0.5 overflow-hidden rounded px-1 py-0.5 text-left text-xs font-medium text-white shadow-sm hover:brightness-90"
							style="{getEventStyle(event)} background-color: {getCalendarColor(event.calendar_id)};"
							onclick={() => onEventClick(event)}
						>
							<span class="block truncate font-semibold">{event.title}</span>
							<span class="block truncate opacity-90">
								{new Date(event.start_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}
							</span>
						</button>
					{/each}
				</div>
			</div>
		{/each}
	</div>
</div>
