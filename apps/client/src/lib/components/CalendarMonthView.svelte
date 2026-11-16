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
	import { eventKey } from '$lib/recurrence';
	import EventAvatar from '$lib/components/EventAvatar.svelte';
	import SourceBadge from '$lib/components/SourceBadge.svelte';
	import { calendarColor, inkOn } from '$lib/calendar-colors';
	import { MONDAY_FIRST, t } from '$lib/i18n/index.svelte';
	import { formatTime, weekdayName } from '$lib/i18n/format';
	import { MonthDrag } from '$lib/month-drag.svelte';

	let {
		events,
		calendars,
		currentDate,
		onDayClick,
		onMoreClick,
		onEventChange,
		onSlotMenu,
		onEventMenu,
		onEventClick
	}: {
		events: AgendaEvent[];
		calendars: CalendarItem[];
		currentDate: CalendarDate;
		onDayClick: (date: CalendarDate) => void;
		onMoreClick: (date: CalendarDate) => void;
		onEventChange: (event: AgendaEvent, start: Date, end: Date) => void;
		onSlotMenu: (e: MouseEvent, start: Date) => void;
		onEventMenu: (e: MouseEvent, event: AgendaEvent) => void;
		onEventClick: (event: AgendaEvent) => void;
	} = $props();

	const dayLabels = $derived(Array.from({ length: 7 }, (_, index) => weekdayName(index, 'short')));
	const tz = getLocalTimeZone();
	const LANE_HEIGHT = 22;
	const MAX_VISIBLE_LANES = 3;
	const DAY_HEADER_OFFSET = 32;
	const CELL_INSET = 6;
	const EVENT_ROW_HEIGHT = 22;
	const CELL_BOTTOM_PAD = 6;
	let cellHeight = $state(0);

	type SpanningSegment = {
		event: AgendaEvent;
		startCol: number;
		span: number;
		lane: number;
		continuesLeft: boolean;
		continuesRight: boolean;
	};

	function canEdit(event: AgendaEvent): boolean {
		if (event.recurrence_rule || event.source) return false;
		return calendars.find((c) => c.id === event.calendar_id)?.role !== 'reader';
	}

	const drag = new MonthDrag({
		canEdit,
		onChange: (event, start, end) => onEventChange(event, start, end)
	});

	function getCalendarColor(calendarId: number): string {
		return calendarColor(calendars, calendarId);
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

	function visibleSingleCount(total: number, lanes: number): number {
		if (cellHeight === 0) return Math.min(total, 3);
		const free = cellHeight - DAY_HEADER_OFFSET - lanes * LANE_HEIGHT - CELL_BOTTOM_PAD;
		const fits = Math.max(0, Math.floor(free / EVENT_ROW_HEIGHT));
		if (total <= fits) return total;
		return Math.max(0, fits - 1);
	}

	function hiddenCountForDay(cell: CalendarDate, segments: SpanningSegment[], total: number, visible: number): number {
		const hiddenSpanning = segments.filter(
			(s) => s.lane >= MAX_VISIBLE_LANES && eventOverlapsDay(s.event, cell)
		).length;
		return hiddenSpanning + (total - visible);
	}

	const todayDate = $derived(today(tz));

	const weeks = $derived(() => {
		const monthStart = startOfMonth(currentDate);
		const monthEnd = endOfMonth(currentDate);
		let gridStart = startOfWeek(monthStart, MONDAY_FIRST);
		let gridEnd = endOfWeek(monthEnd, MONDAY_FIRST);

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

<div class="flex h-full flex-col {drag.active ? 'cursor-grabbing select-none' : ''}">
	<div class="grid grid-cols-7">
		{#each dayLabels as label}
			<div class="py-2 text-center text-xs font-semibold uppercase tracking-wide text-muted-foreground">{label}</div>
		{/each}
	</div>

	<div class="grid flex-1 grid-rows-6">
		{#each weeks() as week, wi (wi)}
			{@const segments = getSpanningSegments(week)}
			{@const lanes = lanesForWeek(segments)}
			<div class="relative grid grid-cols-7">
				{#each week as cell, ci (cell.toString())}
					{@const dayEvts = singleDayEvents(cell)}
					{@const inMonth = isCurrentMonth(cell)}
					{@const today_cell = isToday(cell)}
					{@const visible = visibleSingleCount(dayEvts.length, lanes)}
					{@const hidden = hiddenCountForDay(cell, segments, dayEvts.length, visible)}
					<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
					<div
						role="button"
						tabindex="0"
						data-day={cell.toString()}
						class="group m-0.5 flex cursor-pointer flex-col rounded-fc-md p-1 text-left transition-colors hover:bg-fc-field focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring
							{inMonth ? '' : 'opacity-40'}
							{today_cell ? 'bg-primary/10 ring-2 ring-primary ring-inset' : 'bg-fc-surface'}
							{drag.target === cell.toString() ? 'bg-primary/20! ring-2 ring-primary ring-inset' : ''}"
						bind:clientHeight={
							() => cellHeight,
							(h) => {
								if (wi === 0 && ci === 0) cellHeight = h;
							}
						}
						onclick={() => onDayClick(cell)}
						oncontextmenu={(e) => onSlotMenu(e, new Date(cell.year, cell.month - 1, cell.day, 9))}
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

						<div class="flex min-h-0 flex-col gap-0.5 overflow-hidden">
							{#each dayEvts.slice(0, visible) as event (eventKey(event))}
								{@const fill = getCalendarColor(event.calendar_id)}
								<button
									type="button"
									onpointerdown={(e) => drag.start(e, event)}
									oncontextmenu={(e) => onEventMenu(e, event)}
									class="flex w-full {canEdit(event) ? 'cursor-grab' : 'cursor-pointer'} {drag.isDragging(event) ? 'opacity-40' : ''} items-center gap-1 rounded-md px-1 py-0.5 sm:px-1.5 text-left text-xs font-medium transition-[filter] hover:brightness-90 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
									style="background-color: {fill}; color: {inkOn(fill)};"
									onclick={(e) => {
										e.stopPropagation();
										onEventClick(event);
									}}
								>
									<EventAvatar creator={event.created_by} class="hidden size-3.5 sm:inline-flex" />
									<span class="min-w-0 flex-1 truncate">
										<span class="hidden sm:inline">{formatTime(event.start_at)}{' '}</span>{event.title}
									</span>
									{#if event.recurrence_rule}<iconify-icon icon="solar:repeat-linear" width="12" height="12" aria-label={t('calendar.recurring')} class="block size-3 shrink-0 opacity-80"></iconify-icon>{/if}
								</button>
							{/each}
							{#if hidden > 0}
								<button
									type="button"
									class="w-full cursor-pointer rounded-md px-1 py-0.5 text-left text-xs font-medium text-muted-foreground transition-colors hover:bg-fc-field hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
									onclick={(e) => {
										e.stopPropagation();
										onMoreClick(cell);
									}}
								>
									{t('calendar.moreEvents', { count: hidden })}
								</button>
							{/if}
						</div>
					</div>
				{/each}

				{#each segments as seg}
					{#if seg.lane < MAX_VISIBLE_LANES}
						{@const lPad = seg.continuesLeft ? 0 : CELL_INSET}
						{@const rPad = seg.continuesRight ? 0 : CELL_INSET}
						{@const fill = getCalendarColor(seg.event.calendar_id)}
						<button
							type="button"
							onpointerdown={(e) => drag.start(e, seg.event)}
							oncontextmenu={(e) => onEventMenu(e, seg.event)}
							class="absolute z-10 flex {canEdit(seg.event) ? 'cursor-grab' : 'cursor-pointer'} {drag.isDragging(seg.event) ? 'opacity-40' : ''} items-center truncate px-1.5 text-xs font-medium shadow-sm transition-[filter] hover:brightness-90 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring
								{seg.continuesLeft ? '' : 'rounded-l-md'}
								{seg.continuesRight ? '' : 'rounded-r-md'}"
							style="
								left: calc({seg.startCol} * 100% / 7 + {lPad}px);
								width: calc({seg.span} * 100% / 7 - {lPad + rPad}px);
								top: {DAY_HEADER_OFFSET + seg.lane * LANE_HEIGHT}px;
								height: {LANE_HEIGHT - 2}px;
								background-color: {fill};
								color: {inkOn(fill)};
							"
							onclick={(e) => {
								e.stopPropagation();
								onEventClick(seg.event);
							}}
						>
							<EventAvatar creator={seg.event.created_by} class="mr-1 size-3.5" />
							<span class="mr-1 min-w-0 truncate">{seg.event.title}</span>
							<SourceBadge event={seg.event} class="hidden sm:inline-flex" />
						</button>
					{/if}
				{/each}
			</div>
		{/each}
	</div>
</div>
