<script lang="ts">
	import { getContext } from 'svelte';
	import {
		type CalendarDate,
		today,
		getLocalTimeZone,
		startOfMonth,
		endOfMonth,
		startOfWeek,
		endOfWeek
	} from '@internationalized/date';
	import { backend, type CalendarItem, type AgendaEvent, type CreateEventRequest } from '$lib/backend';
	import { Button } from '$lib/components/ui/button';
	import CalendarMonthView from '$lib/components/CalendarMonthView.svelte';
	import CalendarWeekView from '$lib/components/CalendarWeekView.svelte';
	import CalendarDayView from '$lib/components/CalendarDayView.svelte';
	import CalendarAgendaView from '$lib/components/CalendarAgendaView.svelte';
	import EventModal from '$lib/components/EventModal.svelte';

	type AppContext = { calendars: CalendarItem[]; user: { id: string; email: string; name: string } };
	const app = getContext<AppContext>('app');

	type ViewType = 'month' | 'week' | 'day' | 'agenda';

	const tz = getLocalTimeZone();

	let currentDate = $state<CalendarDate>(today(tz));
	let view = $state<ViewType>('month');
	let events = $state<AgendaEvent[]>([]);
	let loading = $state(false);

	// Modal state
	let modalOpen = $state(false);
	let selectedEvent = $state<AgendaEvent | null>(null);
	let modalInitialDate = $state<string | null>(null);

	// Compute the date range to load based on view
	function getDateRange(): { from: string; to: string } {
		switch (view) {
			case 'month': {
				const ms = startOfMonth(currentDate);
				const me = endOfMonth(currentDate);
				const from = startOfWeek(ms, 'fr-FR').toDate(tz).toISOString();
				const to = endOfWeek(me, 'fr-FR').add({ days: 7 }).toDate(tz).toISOString();
				return { from, to };
			}
			case 'week': {
				const ws = startOfWeek(currentDate, 'fr-FR');
				const we = endOfWeek(currentDate, 'fr-FR');
				return {
					from: ws.toDate(tz).toISOString(),
					to: we.toDate(tz).toISOString()
				};
			}
			case 'day': {
				const from = currentDate.toDate(tz).toISOString();
				const to = currentDate.add({ days: 1 }).toDate(tz).toISOString();
				return { from, to };
			}
			case 'agenda': {
				const from = currentDate.toDate(tz).toISOString();
				const to = currentDate.add({ months: 3 }).toDate(tz).toISOString();
				return { from, to };
			}
		}
	}

	async function loadEvents() {
		const cals = app.calendars;
		if (!cals.length) return;
		loading = true;
		try {
			const { from, to } = getDateRange();
			const results = await Promise.all(
				cals.map((cal) => backend.listEvents(cal.id, from, to))
			);
			events = results.flat();
		} catch {
			events = [];
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		// Reactive dependencies: re-load when currentDate, view, or calendars change
		const _date = currentDate;
		const _view = view;
		const _cals = app.calendars;
		loadEvents();
	});

	// Period title
	const periodTitle = $derived(() => {
		switch (view) {
			case 'month':
				return currentDate.toDate(tz).toLocaleDateString('fr-FR', {
					month: 'long',
					year: 'numeric'
				});
			case 'week': {
				const ws = startOfWeek(currentDate, 'fr-FR').toDate(tz);
				const we = endOfWeek(currentDate, 'fr-FR').toDate(tz);
				const opts: Intl.DateTimeFormatOptions = { day: 'numeric', month: 'short' };
				return `${ws.toLocaleDateString('fr-FR', opts)} – ${we.toLocaleDateString('fr-FR', opts)} ${we.getFullYear()}`;
			}
			case 'day':
				return currentDate.toDate(tz).toLocaleDateString('fr-FR', {
					weekday: 'long',
					day: 'numeric',
					month: 'long',
					year: 'numeric'
				});
			case 'agenda':
				return 'Agenda';
		}
	});

	function navigate(direction: -1 | 1) {
		switch (view) {
			case 'month':
				currentDate = currentDate.add({ months: direction });
				break;
			case 'week':
				currentDate = currentDate.add({ weeks: direction });
				break;
			case 'day':
				currentDate = currentDate.add({ days: direction });
				break;
			case 'agenda':
				currentDate = currentDate.add({ months: direction });
				break;
		}
	}

	function goToday() {
		currentDate = today(tz);
	}

	// Modal handlers
	function openCreateModal(date?: CalendarDate, hour?: number) {
		selectedEvent = null;
		if (date) {
			const d = date.toDate(tz);
			if (hour !== undefined) {
				d.setHours(hour, 0, 0, 0);
			}
			modalInitialDate = d.toISOString().slice(0, 10);
		} else {
			modalInitialDate = null;
		}
		modalOpen = true;
	}

	function openEditModal(ev: AgendaEvent) {
		selectedEvent = ev;
		modalInitialDate = null;
		modalOpen = true;
	}

	async function handleSave(calendarId: number, data: CreateEventRequest) {
		if (selectedEvent) {
			await backend.updateEvent(selectedEvent.id, data);
		} else {
			await backend.createEvent(calendarId, data);
		}
		modalOpen = false;
		await loadEvents();
	}

	async function handleDelete() {
		if (!selectedEvent) return;
		await backend.deleteEvent(selectedEvent.id);
		modalOpen = false;
		await loadEvents();
	}

	function handleClose() {
		modalOpen = false;
		selectedEvent = null;
	}

	function viewVariant(v: ViewType): 'secondary' | 'ghost' {
		return view === v ? 'secondary' : 'ghost';
	}
</script>

<div class="flex h-full flex-col">
	<!-- Top bar -->
	<div class="flex flex-shrink-0 items-center gap-2 border-b px-4 py-2">
		<!-- New event -->
		<Button onclick={() => openCreateModal()}>Nouveau</Button>

		<div class="mx-1 h-5 w-px bg-border"></div>

		<!-- View selector -->
		<div class="flex items-center rounded-lg border">
			<Button variant={viewVariant('month')} size="sm" class="rounded-r-none border-r" onclick={() => (view = 'month')}>
				Mois
			</Button>
			<Button variant={viewVariant('week')} size="sm" class="rounded-none border-r" onclick={() => (view = 'week')}>
				Semaine
			</Button>
			<Button variant={viewVariant('day')} size="sm" class="rounded-none border-r" onclick={() => (view = 'day')}>
				Jour
			</Button>
			<Button variant={viewVariant('agenda')} size="sm" class="rounded-l-none" onclick={() => (view = 'agenda')}>
				Agenda
			</Button>
		</div>

		<div class="mx-1 h-5 w-px bg-border"></div>

		<!-- Navigation -->
		<div class="flex items-center gap-1">
			<Button variant="ghost" size="icon-sm" onclick={() => navigate(-1)} aria-label="Précédent">
				<svg xmlns="http://www.w3.org/2000/svg" class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="15 18 9 12 15 6"/>
				</svg>
			</Button>
			<Button variant="ghost" size="sm" onclick={goToday}>Aujourd'hui</Button>
			<Button variant="ghost" size="icon-sm" onclick={() => navigate(1)} aria-label="Suivant">
				<svg xmlns="http://www.w3.org/2000/svg" class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="9 18 15 12 9 6"/>
				</svg>
			</Button>
		</div>

		<!-- Period title -->
		<span class="ml-2 text-sm font-medium capitalize text-foreground">
			{periodTitle()}
		</span>

		{#if loading}
			<span class="ml-auto text-xs text-muted-foreground">Chargement…</span>
		{/if}
	</div>

	<!-- View area -->
	<div class="min-h-0 flex-1 overflow-hidden">
		{#if view === 'month'}
			<CalendarMonthView
				{events}
				calendars={app.calendars}
				{currentDate}
				onDayClick={(date) => openCreateModal(date)}
				onEventClick={openEditModal}
			/>
		{:else if view === 'week'}
			<CalendarWeekView
				{events}
				calendars={app.calendars}
				{currentDate}
				onSlotClick={(date, hour) => openCreateModal(date, hour)}
				onEventClick={openEditModal}
			/>
		{:else if view === 'day'}
			<CalendarDayView
				{events}
				calendars={app.calendars}
				{currentDate}
				onSlotClick={(hour) => openCreateModal(currentDate, hour)}
				onEventClick={openEditModal}
			/>
		{:else if view === 'agenda'}
			<CalendarAgendaView
				{events}
				calendars={app.calendars}
				onEventClick={openEditModal}
			/>
		{/if}
	</div>
</div>

<!-- Event modal -->
<EventModal
	bind:open={modalOpen}
	event={selectedEvent}
	calendars={app.calendars}
	initialDate={modalInitialDate}
	onSave={handleSave}
	onDelete={handleDelete}
	onClose={handleClose}
/>
