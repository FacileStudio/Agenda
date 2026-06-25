<script lang="ts">
	import {
		type CalendarDate,
		today,
		getLocalTimeZone,
		startOfWeek
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
	const SLOT_HEIGHT = 60;
	const LANE_HEIGHT = 22;
	const DAY_ABBREVS = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim'];

	type SpanningSegment = {
		event: AgendaEvent;
		startCol: number;
		span: number;
		lane: number;
		continuesLeft: boolean;
		continuesRight: boolean;
	};

	function getCalendarColor(calendarId: number): string {
		return calendars.find((c) => c.id === calendarId)?.color ?? '#6b7280';
	}

	const todayDate = $derived(today(tz));

	const weekDays = $derived(() => {
		const weekStart = startOfWeek(currentDate, 'fr-FR');
		return Array.from({ length: 7 }, (_, i) => weekStart.add({ days: i }));
	});

	function toDayStart(d: Date): number {
		return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
	}

	function cellDayStart(cell: CalendarDate): number {
		const d = cell.toDate(tz);
		return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
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
		return toDayStart(new Date(event.start_at)) !== toDayStart(new Date(event.end_at));
	}

	function isSpanning(event: AgendaEvent): boolean {
		return event.is_all_day || isMultiDay(event);
	}

	function eventOverlapsDay(event: AgendaEvent, cell: CalendarDate): boolean {
		const start = new Date(event.start_at).getTime();
		const end = new Date(event.end_at).getTime();
		const dayStart = cellDayStart(cell);
		const dayEnd = dayStart + 86400000;
		return start < dayEnd && end > dayStart;
	}

	const allDaySegments = $derived(() => {
		const days = weekDays();
		const spanning = events.filter((e) => isSpanning(e) && days.some((d) => eventOverlapsDay(e, d)));

		const segments: SpanningSegment[] = [];
		for (const event of spanning) {
			let startCol = -1;
			let endCol = -1;
			for (let i = 0; i < 7; i++) {
				if (eventOverlapsDay(event, days[i])) {
					if (startCol === -1) startCol = i;
					endCol = i;
				}
			}
			if (startCol === -1) continue;

			const eStartDay = toDayStart(new Date(event.start_at));
			const eEndDay = toDayStart(new Date(event.end_at));
			const weekStartDay = cellDayStart(days[0]);
			const weekEndDay = cellDayStart(days[6]);

			segments.push({
				event,
				startCol,
				span: endCol - startCol + 1,
				lane: 0,
				continuesLeft: eStartDay < weekStartDay,
				continuesRight: eEndDay > weekEndDay
			});
		}

		segments.sort((a, b) => a.startCol - b.startCol || b.span - a.span);

		for (let i = 0; i < segments.length; i++) {
			let lane = 0;
			const si = segments[i];
			while (
				segments.some(
					(s, j) =>
						j < i &&
						s.lane === lane &&
						s.startCol < si.startCol + si.span &&
						s.startCol + s.span > si.startCol
				)
			) {
				lane++;
			}
			segments[i].lane = lane;
		}

		return segments;
	});

	const allDayLanes = $derived(() => {
		const segs = allDaySegments();
		if (segs.length === 0) return 0;
		return Math.max(...segs.map((s) => s.lane)) + 1;
	});

	function eventsForDay(cell: CalendarDate): AgendaEvent[] {
		return events.filter((e) => !e.is_all_day && !isMultiDay(e) && isSameDay(e.start_at, cell));
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
</script>

<div class="flex h-full flex-col overflow-hidden">
	<div class="flex-1 overflow-y-auto">
	<!-- Header: day labels + all-day spanning events -->
	<div class="sticky top-0 z-20 border-b bg-background">
		<div class="flex">
			<div class="w-12 flex-shrink-0 border-r"></div>
			<div class="grid flex-1 grid-cols-7">
				{#each weekDays() as day, di (day.toString())}
					<div
						class="flex items-center justify-center gap-1 border-r py-1 text-xs last:border-r-0
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
				{/each}
			</div>
		</div>

		{#if allDaySegments().length > 0}
			<div class="flex border-t">
				<div class="w-12 flex-shrink-0 border-r"></div>
				<div class="relative flex-1" style="height: {allDayLanes() * LANE_HEIGHT + 4}px;">
					{#each allDaySegments() as seg}
						{@const lPad = seg.continuesLeft ? 0 : 2}
						{@const rPad = seg.continuesRight ? 0 : 2}
						<button
							class="absolute z-10 flex cursor-pointer items-center truncate px-1.5 text-xs font-medium text-white shadow-sm transition-[filter] hover:brightness-90
								{seg.continuesLeft ? '' : 'rounded-l'}
								{seg.continuesRight ? '' : 'rounded-r'}"
							style="
								left: calc({seg.startCol} * 100% / 7 + {lPad}px);
								width: calc({seg.span} * 100% / 7 - {lPad + rPad}px);
								top: {seg.lane * LANE_HEIGHT + 2}px;
								height: {LANE_HEIGHT - 2}px;
								background-color: {getCalendarColor(seg.event.calendar_id)};
							"
							onclick={() => onEventClick(seg.event)}
						>
							{seg.event.title}
						</button>
					{/each}
				</div>
			</div>
		{/if}
	</div>

	<!-- Time grid -->
		<div class="flex" style="height: {24 * SLOT_HEIGHT}px;">
			<div class="relative w-12 flex-shrink-0 border-r">
				{#each HOURS as hour}
					<div
						class="absolute left-0 right-0 pr-1 text-right text-xs text-muted-foreground"
						style="top: {hour * SLOT_HEIGHT}px;{hour === 0 ? '' : ' transform: translateY(-50%);'}"
					>
						{formatHour(hour)}
					</div>
				{/each}
			</div>

			{#each weekDays() as day, di (day.toString())}
				<div class="relative flex-1 border-r last:border-r-0">
					{#each HOURS as hour}
						<button
							aria-label={`Creer un evenement a ${hour}h`}
							class="absolute left-0 right-0 cursor-pointer border-b border-dashed border-border/50 hover:bg-accent/30"
							style="top: {hour * SLOT_HEIGHT}px; height: {SLOT_HEIGHT}px;"
							onclick={() => onSlotClick(day, hour)}
						></button>
					{/each}

					{#each eventsForDay(day) as event (event.id)}
						<button
							class="absolute left-0.5 right-0.5 flex cursor-pointer flex-col items-start overflow-hidden rounded-md px-1.5 py-1 text-left text-xs font-medium leading-tight text-white shadow-sm transition-[filter] hover:brightness-90"
							style="{getEventStyle(event)} background-color: {getCalendarColor(event.calendar_id)};"
							onclick={() => onEventClick(event)}
						>
							<span class="block w-full truncate font-semibold">{event.title}</span>
							<span class="mt-0.5 block w-full truncate opacity-90">
								{new Date(event.start_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}
							</span>
						</button>
					{/each}
				</div>
			{/each}
		</div>
	</div>
</div>
