<script lang="ts">
	import {
		type CalendarDate,
		today,
		getLocalTimeZone,
		startOfWeek
	} from '@internationalized/date';
	import type { Activity, AgendaEvent, CalendarItem } from '$lib/backend';
	import { dayStrips, eventGutter, startOfDay } from '$lib/activity-lanes';
	import ActivityStrips from '$lib/components/ActivityStrips.svelte';
	import { eventKey } from '$lib/recurrence';
	import EventAvatar from '$lib/components/EventAvatar.svelte';
	import SourceBadge from '$lib/components/SourceBadge.svelte';
	import { calendarColor, inkOn } from '$lib/calendar-colors';
	import { MONDAY_FIRST, t } from '$lib/i18n/index.svelte';
	import { formatTime, weekdayName } from '$lib/i18n/format';
	import { TimeGridDrag } from '$lib/time-grid-drag.svelte';
	import { atMinutes } from '$lib/week-drag';

	let {
		events,
		activities = [],
		calendars,
		currentDate,
		onSlotClick,
		onRangeCreate,
		onEventChange,
		onSlotMenu,
		onEventMenu,
		onEventClick
	}: {
		events: AgendaEvent[];
		activities?: Activity[];
		calendars: CalendarItem[];
		currentDate: CalendarDate;
		onSlotClick: (date: CalendarDate, hour: number) => void;
		onRangeCreate: (start: Date, end: Date) => void;
		onEventChange: (event: AgendaEvent, start: Date, end: Date) => void;
		onSlotMenu: (e: MouseEvent, start: Date) => void;
		onEventMenu: (e: MouseEvent, event: AgendaEvent) => void;
		onEventClick: (event: AgendaEvent) => void;
	} = $props();

	const tz = getLocalTimeZone();
	const HOURS = Array.from({ length: 24 }, (_, i) => i);
	const SLOT_HEIGHT = 60;
	const LANE_HEIGHT = 22;

	type SpanningSegment = {
		event: AgendaEvent;
		startCol: number;
		span: number;
		lane: number;
		continuesLeft: boolean;
		continuesRight: boolean;
	};

	function getCalendarColor(calendarId: number): string {
		return calendarColor(calendars, calendarId);
	}

	const todayDate = $derived(today(tz));

	const weekDays = $derived(() => {
		const weekStart = startOfWeek(currentDate, MONDAY_FIRST);
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
		return `top: ${top + 1}px; height: ${Math.max(height - 2, 0)}px;`;
	}

	let columns: (HTMLElement | undefined)[] = [];

	function canEdit(event: AgendaEvent): boolean {
		if (event.recurrence_rule || event.source) return false;
		return calendars.find((c) => c.id === event.calendar_id)?.role !== 'reader';
	}

	const drag = new TimeGridDrag({
		columns: () => columns,
		days: () => weekDays().map((d) => d.toDate(tz)),
		slotHeight: SLOT_HEIGHT,
		canEdit,
		onCreate: (start, end) => onRangeCreate(start, end),
		onChange: (event, start, end) => onEventChange(event, start, end)
	});

	function formatHour(h: number): string {
		return h.toString().padStart(2, '0') + ':00';
	}
</script>

<div class="flex h-full flex-col overflow-hidden {drag.active ? 'select-none' : ''}">
	<div class="flex-1 overflow-y-auto">
	<!-- Header: day labels + all-day spanning events -->
	<div class="sticky top-0 z-20 bg-fc-page pb-1">
		<div class="flex">
			<div class="w-12 shrink-0"></div>
			<div class="grid flex-1 grid-cols-7">
				{#each weekDays() as day, di (day.toString())}
					<div
						class="flex items-center justify-center gap-1 py-1 text-xs
							{isToday(day) ? 'font-semibold text-foreground' : 'text-muted-foreground'}"
					>
						{weekdayName(di, 'short')}
						<span
							class="flex size-5 items-center justify-center rounded-full text-xs
								{isToday(day) ? 'bg-primary font-semibold text-primary-foreground' : ''}"
						>
							{day.day}
						</span>
					</div>
				{/each}
			</div>
		</div>

		{#if allDaySegments().length > 0}
			<div class="flex">
				<div class="w-12 shrink-0"></div>
				<div class="relative flex-1" style="height: {allDayLanes() * LANE_HEIGHT + 4}px;">
					{#each allDaySegments() as seg}
						{@const lPad = seg.continuesLeft ? 0 : 2}
						{@const rPad = seg.continuesRight ? 0 : 2}
						{@const fill = getCalendarColor(seg.event.calendar_id)}
						<button
							type="button"
							class="absolute z-10 flex cursor-pointer items-center truncate px-1.5 text-xs font-medium shadow-sm transition-[filter] hover:brightness-90 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring
								{seg.continuesLeft ? '' : 'rounded-l-md'}
								{seg.continuesRight ? '' : 'rounded-r-md'}"
							style="
								left: calc({seg.startCol} * 100% / 7 + {lPad}px);
								width: calc({seg.span} * 100% / 7 - {lPad + rPad}px);
								top: {seg.lane * LANE_HEIGHT + 2}px;
								height: {LANE_HEIGHT - 2}px;
								background-color: {fill};
								color: {inkOn(fill)};
							"
							onclick={() => onEventClick(seg.event)}
						>
							<EventAvatar creator={seg.event.created_by} class="mr-1 size-3.5" />
							<span class="mr-1 min-w-0 truncate">{seg.event.title}</span>
							<SourceBadge event={seg.event} class="hidden sm:inline-flex" />
						</button>
					{/each}
				</div>
			</div>
		{/if}
	</div>

	<!-- Time grid -->
		<div class="flex" style="height: {24 * SLOT_HEIGHT}px;">
			<div class="relative w-12 shrink-0">
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
				{@const strips = dayStrips(activities, startOfDay(day.toDate(tz)), Date.now())}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					bind:this={columns[di]}
					class="relative mx-0.5 flex-1 overflow-hidden rounded-fc-md bg-fc-surface"
					onpointerdown={(e) => drag.startCreate(e, di)}
					oncontextmenu={(e) =>
						onSlotMenu(e, atMinutes(day.toDate(tz), Math.floor(drag.minuteAt(e.clientY, di) / 15) * 15))}
				>
					{#each HOURS as hour}
						<button
							type="button"
							aria-label={t('calendar.createAt', { time: formatTime(new Date(2000, 0, 1, hour)) })}
							class="absolute right-0 left-0 cursor-pointer transition-colors hover:bg-fc-field focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring {hour % 2 === 1 ? 'bg-fc-field/50' : ''}"
							style="top: {hour * SLOT_HEIGHT}px; height: {SLOT_HEIGHT}px;"
							onclick={() => onSlotClick(day, hour)}
						></button>
					{/each}

					{#each eventsForDay(day) as event (eventKey(event))}
						{@const fill = getCalendarColor(event.calendar_id)}
						{@const editable = canEdit(event)}
						<button
							type="button"
							onpointerdown={(e) => drag.startEvent(e, event, 'move', di)}
							oncontextmenu={(e) => onEventMenu(e, event)}
							class="absolute left-0.5 flex {editable ? 'cursor-grab' : 'cursor-pointer'} {drag.isDragging(event) ? 'opacity-40' : ''} flex-col items-start overflow-hidden rounded-md px-1.5 py-1 text-left text-xs leading-tight font-medium shadow-sm transition-[filter] hover:brightness-90 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
							style="{getEventStyle(event)} right: {eventGutter(strips, event.start_at, event.end_at)}px; background-color: {fill}; color: {inkOn(fill)};"
							onclick={() => onEventClick(event)}
						>
							{#if editable}
								<span
									role="presentation"
									class="absolute inset-x-0 top-0 z-10 h-1.5 cursor-ns-resize"
									onpointerdown={(e) => drag.startEvent(e, event, 'resize-top', di)}
								></span>
								<span
									role="presentation"
									class="absolute inset-x-0 bottom-0 z-10 h-1.5 cursor-ns-resize"
									onpointerdown={(e) => drag.startEvent(e, event, 'resize-bottom', di)}
								></span>
							{/if}
							<span class="flex w-full items-center gap-1">
								<span class="min-w-0 flex-1 truncate font-semibold">{event.title}</span>
								{#if event.recurrence_rule}<iconify-icon icon="solar:repeat-linear" width="12" height="12" aria-label={t('calendar.recurring')} class="block size-3 shrink-0 opacity-80"></iconify-icon>{/if}
								<EventAvatar creator={event.created_by} />
							</span>
							<span class="mt-0.5 block w-full truncate opacity-90">
								{formatTime(event.start_at)}
							</span>
						</button>
					{/each}

					<ActivityStrips {activities} day={day.toDate(tz)} slotHeight={SLOT_HEIGHT} />

					{#if drag.preview && drag.preview.column === di}
						{@const range = drag.preview.range}
						{@const ghostFill = drag.gesture?.event ? getCalendarColor(drag.gesture.event.calendar_id) : null}
						<div
							class="pointer-events-none absolute right-2 left-0.5 z-30 overflow-hidden rounded-md px-1.5 py-1 text-xs leading-tight font-semibold shadow-md ring-2 ring-primary"
							style="top: {(range.start / 60) * SLOT_HEIGHT + 1}px; height: {((range.end - range.start) / 60) * SLOT_HEIGHT - 2}px; background-color: {ghostFill ?? 'var(--color-primary)'}; color: {ghostFill ? inkOn(ghostFill) : 'var(--color-primary-foreground)'};"
						>
							{formatTime(atMinutes(day.toDate(tz), range.start))} – {formatTime(atMinutes(day.toDate(tz), range.end))}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	</div>
</div>
