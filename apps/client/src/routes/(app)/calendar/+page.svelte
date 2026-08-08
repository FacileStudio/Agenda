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
	import { Button, IconButton, Spinner, icons } from '@facile/muse';
	import { backend, type CalendarItem, type AgendaEvent, type CreateEventRequest } from '$lib/backend';
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
			const results = await Promise.all(cals.map((cal) => backend.listEvents(cal.id, from, to)));
			events = results.flat();
		} catch {
			events = [];
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		const _date = currentDate;
		const _view = view;
		const _cals = app.calendars;
		loadEvents();
	});

	const periodTitle = $derived(() => {
		switch (view) {
			case 'month':
				return currentDate.toDate(tz).toLocaleDateString('fr-FR', { month: 'long', year: 'numeric' });
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
			case 'month': currentDate = currentDate.add({ months: direction }); break;
			case 'week': currentDate = currentDate.add({ weeks: direction }); break;
			case 'day': currentDate = currentDate.add({ days: direction }); break;
			case 'agenda': currentDate = currentDate.add({ months: direction }); break;
		}
	}

	function goToday() { currentDate = today(tz); }

	function openCreateModal(date?: CalendarDate, hour?: number) {
		selectedEvent = null;
		if (date) {
			modalInitialDate = `${date.year}-${String(date.month).padStart(2, '0')}-${String(date.day).padStart(2, '0')}`;
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
			await backend.updateEvent(selectedEvent.id, calendarId, data);
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

	const viewItems: { id: ViewType; label: string; icon: string }[] = [
		{ id: 'month', label: 'Mois', icon: 'solar:calendar-minimalistic-linear' },
		{ id: 'week', label: 'Semaine', icon: icons.calendar },
		{ id: 'day', label: 'Jour', icon: 'solar:calendar-date-linear' },
		{ id: 'agenda', label: 'Agenda', icon: 'solar:list-linear' }
	];
</script>

<div class="flex h-full flex-col">
	<!-- Top bar -->
	<div class="flex h-16 shrink-0 items-center gap-2 border-b border-border px-3 sm:gap-3 sm:px-5">
		<!-- Left: today + navigation + period title -->
		<div class="flex min-w-0 flex-1 items-center gap-2 sm:gap-3">
			<Button
				variant="outline"
				size="lg"
				icon="solar:calendar-minimalistic-linear"
				class="hidden sm:inline-flex"
				onclick={goToday}
			>
				Aujourd'hui
			</Button>
			<IconButton class="sm:hidden" onclick={goToday} aria-label="Aujourd'hui">
				<iconify-icon
					icon="solar:calendar-minimalistic-linear"
					width="18"
					height="18"
					class="block size-4.5"
				></iconify-icon>
			</IconButton>

			<div class="flex items-center">
				<IconButton variant="ghost" onclick={() => navigate(-1)} aria-label="Précédent">
					<iconify-icon icon={icons.chevronLeft} width="20" height="20" class="block size-5"
					></iconify-icon>
				</IconButton>
				<IconButton variant="ghost" onclick={() => navigate(1)} aria-label="Suivant">
					<iconify-icon icon={icons.arrow} width="20" height="20" class="block size-5"></iconify-icon>
				</IconButton>
			</div>

			<h1 class="min-w-0 truncate text-fc-lg font-semibold capitalize text-foreground sm:text-fc-xl">
				{periodTitle()}
			</h1>

			{#if loading}
				<Spinner size="sm" class="shrink-0" label="Chargement" />
			{/if}
		</div>

		<!-- Right: view switcher + new event -->
		<div class="flex shrink-0 items-center gap-2 sm:gap-3">
			<div class="flex min-w-0 items-center gap-1 rounded-fc-pill bg-muted p-1">
				{#each viewItems as item}
					{@const active = view === item.id}
					<button
						type="button"
						onclick={() => (view = item.id)}
						aria-pressed={active}
						aria-label={item.label}
						class="flex h-9 shrink-0 cursor-pointer items-center gap-1.5 rounded-fc-pill px-3 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring {active
							? 'bg-primary text-primary-foreground'
							: 'text-muted-foreground hover:text-foreground'}"
					>
						<iconify-icon icon={item.icon} width="16" height="16" class="block size-4 shrink-0"
						></iconify-icon>
						<span class="hidden md:inline">{item.label}</span>
					</button>
				{/each}
			</div>

			<Button
				onclick={() => openCreateModal()}
				size="lg"
				icon={icons.plus}
				aria-label="Nouvel événement"
			>
				<span class="hidden sm:inline">Nouveau</span>
			</Button>
		</div>
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
