<script lang="ts">
	import {
		Alert,
		Avatar,
		Button,
		Checkbox,
		ConfirmModal,
		Field,
		Input,
		Modal,
		Select,
		Textarea,
		icons
	} from '@facile/muse';
	import { resolveFileUrl, type AgendaEvent, type CalendarItem, type CreateEventRequest } from '$lib/backend';
	import { roomName } from '$lib/room-name';

	let {
		open = $bindable(false),
		event,
		calendars,
		initialDate,
		onSave,
		onDelete,
		onClose
	}: {
		open: boolean;
		event: AgendaEvent | null;
		calendars: CalendarItem[];
		initialDate: string | null;
		onSave: (calendarId: number, data: CreateEventRequest) => Promise<void>;
		onDelete: () => Promise<void>;
		onClose: () => void;
	} = $props();

	function splitDT(iso: string) {
		return { date: iso.slice(0, 10), time: iso.slice(11, 16) || '09:00' };
	}

	// Build a LOCAL wall-clock "YYYY-MM-DDTHH:mm" string. The stored start_at/
	// end_at are UTC instants; toISOString() would yield UTC wall-clock (e.g.
	// 11:00 for a 13:00 CEST event), so we read local components instead. This
	// matches the save path, which parses the inputs as local time.
	function pad(n: number): string {
		return String(n).padStart(2, '0');
	}
	function toLocalInput(d: Date): string {
		return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
	}

	function defaultStart(): string {
		if (event) return toLocalInput(new Date(event.start_at));
		if (initialDate) return `${initialDate}T09:00`;
		const d = new Date();
		d.setMinutes(0, 0, 0);
		return toLocalInput(d);
	}

	function defaultEnd(): string {
		if (event) return toLocalInput(new Date(event.end_at));
		if (initialDate) return `${initialDate}T10:00`;
		const d = new Date();
		d.setMinutes(0, 0, 0);
		d.setHours(d.getHours() + 1);
		return toLocalInput(d);
	}

	let title = $state('');
	let description = $state('');
	let location = $state('');
	let isAllDay = $state(false);
	let status = $state('confirmed');
	let calendarId = $state(0);
	let saving = $state(false);
	let deleting = $state(false);
	let confirmDeleteOpen = $state(false);
	let error = $state('');
	let conferenceUrl = $state('');
	let conferenceProvider = $state('');

	let startDateStr = $state('');
	let startTimeStr = $state('');
	let endDateStr = $state('');
	let endTimeStr = $state('');
	let durationMs = $state(60 * 60 * 1000);

	$effect(() => {
		if (open) {
			title = event?.title ?? '';
			description = event?.description ?? '';
			location = event?.location ?? '';
			isAllDay = event?.is_all_day ?? false;
			status = event?.status ?? 'confirmed';
			calendarId = event?.calendar_id ?? (calendars[0]?.id ?? 0);
			conferenceUrl = event?.conference_url ?? '';
			conferenceProvider = event?.conference_provider ?? '';
			error = '';
			confirmDeleteOpen = false;
			const s = splitDT(defaultStart());
			const e = splitDT(defaultEnd());
			startDateStr = s.date;
			startTimeStr = s.time;
			endDateStr = e.date;
			endTimeStr = e.time;
			const allDay = event?.is_all_day ?? false;
			const startAnchor = new Date(`${s.date}T${allDay ? '00:00' : (s.time || '00:00')}`);
			const endAnchor = new Date(`${e.date}T${allDay ? '00:00' : (e.time || '00:00')}`);
			const diff = endAnchor.getTime() - startAnchor.getTime();
			durationMs = (allDay ? diff >= 0 : diff > 0) ? diff : 60 * 60 * 1000;
		}
	});

	// Wall-clock anchor for duration maths. All-day events are measured midnight
	// to midnight (day granularity); timed events use their real times.
	function anchorDT(dateStr: string, timeStr: string): Date {
		return new Date(`${dateStr}T${isAllDay ? '00:00' : (timeStr || '00:00')}`);
	}

	// Current valid duration in ms, or null if the range is invalid. All-day
	// allows a zero-length (single-day) span; timed events must be strictly > 0.
	function currentDuration(): number | null {
		const start = anchorDT(startDateStr, startTimeStr);
		const end = anchorDT(endDateStr, endTimeStr);
		if (isNaN(start.getTime()) || isNaN(end.getTime())) return null;
		const diff = end.getTime() - start.getTime();
		return (isAllDay ? diff >= 0 : diff > 0) ? diff : null;
	}

	// Start moved: shift the end to preserve the event's duration (Google/Apple
	// Calendar behaviour). Auto-resolves a start dragged past the old end, for
	// both timed and multi-day all-day events. All-day uses calendar-day
	// arithmetic so the span survives DST and month boundaries.
	function shiftEndToPreserveDuration() {
		if (isAllDay) {
			const start = new Date(`${startDateStr}T00:00`);
			if (isNaN(start.getTime())) return;
			const days = Math.round(durationMs / 86400000);
			const end = new Date(start.getFullYear(), start.getMonth(), start.getDate() + days);
			endDateStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}`;
			return;
		}
		const start = anchorDT(startDateStr, startTimeStr);
		if (isNaN(start.getTime())) return;
		const end = new Date(start.getTime() + durationMs);
		endDateStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}`;
		endTimeStr = `${pad(end.getHours())}:${pad(end.getMinutes())}`;
	}

	// End moved: learn the new duration, or snap the end back if the user picked
	// an end before the start so the form never holds an invalid range.
	function rememberDurationFromEnd() {
		const d = currentDuration();
		if (d === null) shiftEndToPreserveDuration();
		else durationMs = d;
	}

	const selectedCalendar = $derived(calendars.find((c) => c.id === calendarId));
	const echoAvailable = $derived(!!selectedCalendar?.echo_url);

	function toggleConference() {
		if (conferenceUrl) {
			conferenceUrl = '';
			conferenceProvider = '';
		} else if (selectedCalendar?.echo_url) {
			const uid = event?.uid ?? crypto.randomUUID().replace(/-/g, '');
			conferenceUrl = `${selectedCalendar.echo_url.replace(/\/$/, '')}/${roomName(uid)}`;
			conferenceProvider = 'Echo';
		}
	}

	async function handleSave() {
		if (!title.trim()) { error = 'Le titre est requis.'; return; }
		if (!calendarId || !calendars.find((c) => c.id === calendarId)) {
			error = 'Selectionnez un calendrier valide.';
			return;
		}
		const startISO = `${startDateStr}T${isAllDay ? '00:00' : (startTimeStr || '00:00')}`;
		const endISO = `${endDateStr}T${isAllDay ? '23:59' : (endTimeStr || '00:00')}`;
		if (!isAllDay) {
			const s = new Date(startISO);
			const e = new Date(endISO);
			if (!isNaN(s.getTime()) && !isNaN(e.getTime()) && e <= s) {
				error = 'La fin doit etre apres le debut.';
				return;
			}
		}
		saving = true;
		error = '';
		try {
			const data: CreateEventRequest = {
				title: title.trim(),
				description: description || undefined,
				location: location || undefined,
				start_at: new Date(startISO).toISOString(),
				end_at: new Date(endISO).toISOString(),
				is_all_day: isAllDay,
				status: status || undefined,
				conference_url: conferenceUrl || undefined,
				conference_provider: conferenceProvider || undefined
			};
			await onSave(calendarId, data);
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Une erreur est survenue.';
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		deleting = true;
		error = '';
		try {
			await onDelete();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Une erreur est survenue.';
			throw e;
		} finally {
			deleting = false;
		}
	}
</script>

<Modal
	bind:open
	size="lg"
	showClose
	{onClose}
	title={event ? "Modifier l'evenement" : 'Nouvel evenement'}
>
	<div class="flex flex-col gap-4">
		<Field label="Titre">
			<Input type="text" placeholder="Titre de l'evenement" bind:value={title} autofocus />
		</Field>

		<Field label="Calendrier">
			<div class="flex items-center gap-2">
				{#if selectedCalendar}
					<span
						class="size-2.5 shrink-0 rounded-full"
						style="background-color: {selectedCalendar.color}"
					></span>
				{/if}
				<Select
					value={String(calendarId)}
					onchange={(e) => (calendarId = Number(e.currentTarget.value))}
				>
					{#each calendars as cal (cal.id)}
						<option value={String(cal.id)}>{cal.name}</option>
					{/each}
				</Select>
			</div>
		</Field>

		<Checkbox bind:checked={isAllDay} label="Toute la journee" />

		<Field label="Debut">
			<div class="flex gap-2">
				<Input
					type="date"
					bind:value={startDateStr}
					onchange={shiftEndToPreserveDuration}
					class="flex-1"
				/>
				{#if !isAllDay}
					<Input
						type="time"
						aria-label="Heure de debut"
						bind:value={startTimeStr}
						onchange={shiftEndToPreserveDuration}
						class="w-32 shrink-0"
					/>
				{/if}
			</div>
		</Field>

		<Field label="Fin">
			<div class="flex gap-2">
				<Input
					type="date"
					bind:value={endDateStr}
					onchange={rememberDurationFromEnd}
					class="flex-1"
				/>
				{#if !isAllDay}
					<Input
						type="time"
						aria-label="Heure de fin"
						bind:value={endTimeStr}
						onchange={rememberDurationFromEnd}
						class="w-32 shrink-0"
					/>
				{/if}
			</div>
		</Field>

		<Field label="Lieu">
			<Input type="text" placeholder="Lieu (optionnel)" bind:value={location} />
		</Field>

		{#if echoAvailable || conferenceUrl}
			<div class="flex flex-col gap-1.5">
				<span class="text-fc-sm text-fc-fg">Visioconference</span>
				{#if conferenceUrl}
					<div class="flex items-center gap-2 rounded-fc-md bg-fc-surface px-3 py-2">
						<iconify-icon
							icon="solar:videocamera-record-linear"
							width="18"
							height="18"
							class="block size-4.5 shrink-0 text-fc-fg-muted"
						></iconify-icon>
						<a
							href={conferenceUrl}
							target="_blank"
							rel="noopener"
							class="min-w-0 flex-1 truncate text-fc-sm text-fc-fg underline underline-offset-2"
						>
							{conferenceUrl}
						</a>
						<Button
							variant="ghost"
							size="sm"
							icon={icons.close}
							aria-label="Retirer la visioconference"
							onclick={toggleConference}
						/>
					</div>
				{:else}
					<Button
						variant="outline"
						icon="solar:videocamera-record-linear"
						class="w-full justify-start"
						onclick={toggleConference}
					>
						Ajouter une visio Echo
					</Button>
				{/if}
			</div>
		{/if}

		<Field label="Description">
			<Textarea placeholder="Description (optionnelle)" bind:value={description} rows={3} />
		</Field>

		<Field label="Statut">
			<Select bind:value={status}>
				<option value="confirmed">Confirme</option>
				<option value="tentative">Provisoire</option>
				<option value="cancelled">Annule</option>
			</Select>
		</Field>

		{#if event?.created_by}
			{@const creator = event.created_by}
			<div class="flex items-center gap-2 border-t border-fc-border pt-3 text-fc-sm text-fc-fg-muted">
				<Avatar
					size="sm"
					class="size-6 text-fc-xs"
					src={creator.avatar_url ? resolveFileUrl(creator.avatar_url) : undefined}
					name={creator.name || creator.email}
				/>
				<span>Cree par <span class="font-medium text-fc-fg">{creator.name || creator.email}</span></span>
			</div>
		{/if}

		{#if error}
			<Alert tone="danger">{error}</Alert>
		{/if}
	</div>

	{#snippet footer()}
		<div class="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between">
			{#if event}
				<Button
					variant="danger"
					icon={icons.remove}
					class="w-full sm:w-auto"
					onclick={() => (confirmDeleteOpen = true)}
					disabled={deleting || saving}
				>
					{deleting ? 'Suppression…' : 'Supprimer'}
				</Button>
			{:else}
				<span class="hidden sm:block"></span>
			{/if}
			<div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
				<Button
					variant="outline"
					class="w-full sm:w-auto"
					onclick={() => (open = false)}
					disabled={saving || deleting}
				>
					Annuler
				</Button>
				<Button
					icon={icons.check}
					class="w-full sm:w-auto"
					onclick={handleSave}
					disabled={saving || deleting}
				>
					{saving ? 'Enregistrement…' : 'Enregistrer'}
				</Button>
			</div>
		</div>
	{/snippet}
</Modal>

<ConfirmModal
	bind:open={confirmDeleteOpen}
	tone="danger"
	title="Supprimer cet evenement ?"
	description={`« ${title || 'Cet evenement'} » disparaitra de ce calendrier pour tout le monde, et des clients CalDAV deja synchronises. Cette action est irreversible.`}
	confirmLabel="Supprimer"
	onConfirm={handleDelete}
/>
