<script lang="ts">
	import {
		type CalendarDate,
		today,
		getLocalTimeZone,
		startOfMonth,
		endOfMonth,
		startOfWeek,
		endOfWeek
	} from '@internationalized/date';
	import type { AgendaEvent, CalendarItem } from '$lib/backend';

	let {
		events,
		calendars,
		currentDate,
		onDayClick,
		onEventClick
	}: {
		events: AgendaEvent[];
		calendars: CalendarItem[];
		currentDate: CalendarDate;
		onDayClick: (date: CalendarDate) => void;
		onEventClick: (event: AgendaEvent) => void;
	} = $props();

	const DAY_LABELS = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim'];
	const tz = getLocalTimeZone();

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

	function eventsForDay(cell: CalendarDate): AgendaEvent[] {
		return events.filter((e) => isSameDay(e.start_at, cell));
	}

	const todayDate = $derived(today(tz));

	const weeks = $derived(() => {
		const monthStart = startOfMonth(currentDate);
		const monthEnd = endOfMonth(currentDate);
		// Grid starts on Monday before/at month start
		let gridStart = startOfWeek(monthStart, 'fr-FR');
		let gridEnd = endOfWeek(monthEnd, 'fr-FR');

		// Ensure we have exactly 6 weeks
		const days: CalendarDate[] = [];
		let cursor = gridStart;
		while (cursor.compare(gridEnd) <= 0) {
			days.push(cursor);
			cursor = cursor.add({ days: 1 });
		}
		// Pad to 42 days (6 weeks)
		while (days.length < 42) {
			days.push(days[days.length - 1].add({ days: 1 }));
		}
		const result: CalendarDate[][] = [];
		for (let i = 0; i < 6; i++) {
			result.push(days.slice(i * 7, i * 7 + 7));
		}
		return result;
	});

	function isToday(cell: CalendarDate): boolean {
		return cell.compare(todayDate) === 0;
	}

	function isCurrentMonth(cell: CalendarDate): boolean {
		return cell.month === currentDate.month && cell.year === currentDate.year;
	}
</script>

<div class="flex h-full flex-col">
	<!-- Day headers -->
	<div class="grid grid-cols-7 border-b">
		{#each DAY_LABELS as label}
			<div class="py-2 text-center text-xs font-medium text-muted-foreground">{label}</div>
		{/each}
	</div>

	<!-- Calendar grid -->
	<div class="grid flex-1 grid-rows-6">
		{#each weeks() as week, wi (wi)}
			<div class="grid grid-cols-7 border-b last:border-b-0">
				{#each week as cell (cell.toString())}
					{@const dayEvents = eventsForDay(cell)}
					{@const inMonth = isCurrentMonth(cell)}
					{@const today_cell = isToday(cell)}
					<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
					<div
						role="button"
						tabindex="0"
						class="group relative flex flex-col border-r p-1 text-left transition-colors last:border-r-0 hover:bg-accent/50 cursor-pointer
							{inMonth ? '' : 'opacity-40'}"
						onclick={() => onDayClick(cell)}
						onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') onDayClick(cell); }}
					>
						<!-- Day number -->
						<span
							class="mb-1 flex size-6 items-center justify-center self-start rounded-full text-xs font-medium
								{today_cell
									? 'bg-primary text-primary-foreground'
									: inMonth
										? 'text-foreground'
										: 'text-muted-foreground'}"
						>
							{cell.day}
						</span>

						<!-- Events -->
						<div class="flex flex-col gap-0.5 overflow-hidden">
							{#each dayEvents.slice(0, 3) as event (event.id)}
								<button
									class="w-full truncate rounded px-1 py-0.5 text-left text-xs font-medium text-white"
									style="background-color: {getCalendarColor(event.calendar_id)}"
									onclick={(e) => {
										e.stopPropagation();
										onEventClick(event);
									}}
								>
									{event.is_all_day ? '' : new Date(event.start_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' }) + ' '}
									{event.title}
								</button>
							{/each}
							{#if dayEvents.length > 3}
								<span class="px-1 text-xs text-muted-foreground">
									+{dayEvents.length - 3} de plus
								</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/each}
	</div>
</div>
