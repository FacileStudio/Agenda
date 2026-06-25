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
	const LANE_HEIGHT = 22;
	const MAX_VISIBLE_LANES = 3;
	const DAY_HEADER_OFFSET = 30;

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

	function toDayStart(d: Date): number {
		return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
	}

	function cellDayStart(cell: CalendarDate): number {
		const d = cell.toDate(tz);
		return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
	}

	function isSameDay(dateStr: string, cell: CalendarDate): boolean {
		const d = new Date(dateStr);
		return d.getFullYear() === cell.year && d.getMonth() + 1 === cell.month && d.getDate() === cell.day;
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

	function getSpanningSegments(week: CalendarDate[]): SpanningSegment[] {
		const spanning = events.filter((e) => isSpanning(e) && week.some((d) => eventOverlapsDay(e, d)));

		const segments: SpanningSegment[] = [];
		for (const event of spanning) {
			let startCol = -1;
			let endCol = -1;
			for (let i = 0; i < 7; i++) {
				if (eventOverlapsDay(event, week[i])) {
					if (startCol === -1) startCol = i;
					endCol = i;
				}
			}
			if (startCol === -1) continue;

			const eStartDay = toDayStart(new Date(event.start_at));
			const eEndDay = toDayStart(new Date(event.end_at));
			const weekStartDay = cellDayStart(week[0]);
			const weekEndDay = cellDayStart(week[6]);

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
	}

	function singleDayEvents(cell: CalendarDate): AgendaEvent[] {
		return events.filter((e) => !isSpanning(e) && isSameDay(e.start_at, cell));
	}

	function lanesForWeek(segments: SpanningSegment[]): number {
		if (segments.length === 0) return 0;
		return Math.min(Math.max(...segments.map((s) => s.lane)) + 1, MAX_VISIBLE_LANES);
	}

	function hiddenCountForDay(cell: CalendarDate, segments: SpanningSegment[], dayEvts: AgendaEvent[]): number {
		const hiddenSpanning = segments.filter(
			(s) => s.lane >= MAX_VISIBLE_LANES && eventOverlapsDay(s.event, cell)
		).length;
		const hiddenSingle = Math.max(0, dayEvts.length - 3);
		return hiddenSpanning + hiddenSingle;
	}

	const todayDate = $derived(today(tz));

	const weeks = $derived(() => {
		const monthStart = startOfMonth(currentDate);
		const monthEnd = endOfMonth(currentDate);
		let gridStart = startOfWeek(monthStart, 'fr-FR');
		let gridEnd = endOfWeek(monthEnd, 'fr-FR');

		const days: CalendarDate[] = [];
		let cursor = gridStart;
		while (cursor.compare(gridEnd) <= 0) {
			days.push(cursor);
			cursor = cursor.add({ days: 1 });
		}
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
	<div class="grid grid-cols-7 border-b">
		{#each DAY_LABELS as label}
			<div class="py-2 text-center text-xs font-semibold uppercase tracking-wide text-muted-foreground">{label}</div>
		{/each}
	</div>

	<div class="grid flex-1 grid-rows-6">
		{#each weeks() as week, wi (wi)}
			{@const segments = getSpanningSegments(week)}
			{@const lanes = lanesForWeek(segments)}
			<div class="relative grid grid-cols-7 border-b last:border-b-0">
				{#each week as cell, ci (cell.toString())}
					{@const dayEvts = singleDayEvents(cell)}
					{@const inMonth = isCurrentMonth(cell)}
					{@const today_cell = isToday(cell)}
					{@const hidden = hiddenCountForDay(cell, segments, dayEvts)}
					<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
					<div
						role="button"
						tabindex="0"
						class="group flex flex-col border-r p-1 text-left transition-colors last:border-r-0 hover:bg-accent/50 cursor-pointer
							{inMonth ? '' : 'opacity-40'}"
						onclick={() => onDayClick(cell)}
						onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') onDayClick(cell); }}
					>
						<span
							class="mb-0.5 flex size-6 items-center justify-center self-start rounded-full text-xs font-medium
								{today_cell
									? 'bg-primary text-primary-foreground'
									: inMonth
										? 'text-foreground'
										: 'text-muted-foreground'}"
						>
							{cell.day}
						</span>

						{#if lanes > 0}
							<div style="height: {lanes * LANE_HEIGHT}px;"></div>
						{/if}

						<div class="flex flex-col gap-0.5 overflow-hidden">
							{#each dayEvts.slice(0, 3) as event (event.id)}
								<button
									class="w-full cursor-pointer truncate rounded-md px-1.5 py-0.5 text-left text-xs font-medium text-white transition-[filter] hover:brightness-90"
									style="background-color: {getCalendarColor(event.calendar_id)}"
									onclick={(e) => {
										e.stopPropagation();
										onEventClick(event);
									}}
								>
									{new Date(event.start_at).toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}
									{' '}{event.title}
								</button>
							{/each}
							{#if hidden > 0}
								<span class="px-1 text-xs text-muted-foreground">
									+{hidden} de plus
								</span>
							{/if}
						</div>
					</div>
				{/each}

				{#each segments as seg}
					{#if seg.lane < MAX_VISIBLE_LANES}
						{@const lPad = seg.continuesLeft ? 0 : 2}
						{@const rPad = seg.continuesRight ? 0 : 2}
						<button
							class="absolute z-10 flex cursor-pointer items-center truncate px-1.5 text-xs font-medium text-white shadow-sm transition-[filter] hover:brightness-90
								{seg.continuesLeft ? '' : 'rounded-l'}
								{seg.continuesRight ? '' : 'rounded-r'}"
							style="
								left: calc({seg.startCol} * 100% / 7 + {lPad}px);
								width: calc({seg.span} * 100% / 7 - {lPad + rPad}px);
								top: {DAY_HEADER_OFFSET + seg.lane * LANE_HEIGHT}px;
								height: {LANE_HEIGHT - 2}px;
								background-color: {getCalendarColor(seg.event.calendar_id)};
							"
							onclick={(e) => {
								e.stopPropagation();
								onEventClick(seg.event);
							}}
						>
							{seg.event.title}
						</button>
					{/if}
				{/each}
			</div>
		{/each}
	</div>
</div>
