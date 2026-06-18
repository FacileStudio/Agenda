<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import * as Select from '$lib/components/ui/select';
	import * as Popover from '$lib/components/ui/popover';
	import { Calendar } from '$lib/components/ui/calendar';
	import { parseDate, type DateValue } from '@internationalized/date';
	import { cn } from '$lib/utils';
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

	let title = $state(event?.title ?? '');
	let description = $state(event?.description ?? '');
	let location = $state(event?.location ?? '');
	let isAllDay = $state(event?.is_all_day ?? false);
	let status = $state(event?.status ?? 'confirmed');
	let calendarId = $state<number>(event?.calendar_id ?? (calendars[0]?.id ?? 0));
	let saving = $state(false);
	let deleting = $state(false);
	let error = $state('');
	let conferenceUrl = $state('');
	let conferenceProvider = $state('');

	let startDateStr = $state('');
	let startTimeStr = $state('');
	let endDateStr = $state('');
	let endTimeStr = $state('');
	let startPickerOpen = $state(false);
	let endPickerOpen = $state(false);

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
			const s = splitDT(defaultStart());
			const e = splitDT(defaultEnd());
			startDateStr = s.date;
			startTimeStr = s.time;
			endDateStr = e.date;
			endTimeStr = e.time;
		}
	});

	function safeParseDate(dateStr: string): DateValue | undefined {
		try { return parseDate(dateStr); } catch { return undefined; }
	}

	const startDateVal = $derived(safeParseDate(startDateStr));
	const endDateVal = $derived(safeParseDate(endDateStr));
	const selectedCalendar = $derived(calendars.find(c => c.id === calendarId));
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

	function formatDisplayDate(dateStr: string): string {
		if (!dateStr) return 'Choisir une date';
		const d = new Date(dateStr + 'T12:00');
		return d.toLocaleDateString('fr-FR', {
			weekday: 'short',
			day: 'numeric',
			month: 'long',
			year: 'numeric'
		});
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
		if (!confirm('Supprimer cet evenement ?')) return;
		deleting = true;
		error = '';
		try {
			await onDelete();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Une erreur est survenue.';
		} finally {
			deleting = false;
		}
	}

	function handleOpenChange(val: boolean) {
		if (!val) onClose();
	}
</script>

<DialogPrimitive.Root bind:open onOpenChange={handleOpenChange}>
	<DialogPrimitive.Portal>
		<DialogPrimitive.Overlay
			class="data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 fixed inset-0 z-50 bg-black/40 supports-backdrop-filter:backdrop-blur-xs"
		/>
		<DialogPrimitive.Content
			class={cn(
				'data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95',
				'fixed top-1/2 left-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2',
				'rounded-2xl border bg-background p-6 shadow-xl'
			)}
		>
			<!-- Header -->
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">
					{event ? "Modifier l'evenement" : 'Nouvel evenement'}
				</h2>
				<DialogPrimitive.Close
					class="cursor-pointer rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
				>
					<iconify-icon icon="solar:close-circle-linear" width="18"></iconify-icon>
				</DialogPrimitive.Close>
			</div>

			<!-- Form -->
			<div class="flex flex-col gap-4">
				<!-- Title -->
				<div class="flex flex-col gap-1.5">
					<Label for="event-title">Titre</Label>
					<Input
						id="event-title"
						type="text"
						placeholder="Titre de l'evenement"
						bind:value={title}
						autofocus
					/>
				</div>

				<!-- Calendar -->
				<div class="flex flex-col gap-1.5">
					<Label>Calendrier</Label>
					<Select.Root
						type="single"
						value={String(calendarId)}
						onValueChange={(v: string) => { calendarId = Number(v); }}
					>
						<Select.Trigger class="w-full">
							<span class="flex items-center gap-2">
								{#if calendars.find(c => c.id === calendarId)}
									<span
										class="inline-block size-2.5 rounded-full"
										style="background-color: {calendars.find(c => c.id === calendarId)?.color}"
									></span>
								{/if}
								{calendars.find((c) => c.id === calendarId)?.name ?? 'Choisir un calendrier'}
							</span>
						</Select.Trigger>
						<Select.Content>
							{#each calendars as cal (cal.id)}
								<Select.Item value={String(cal.id)}>
									<span
										class="mr-2 inline-block size-2.5 rounded-full"
										style="background-color: {cal.color}"
									></span>
									{cal.name}
								</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>

				<!-- All day toggle -->
				<div class="flex items-center gap-2">
					<Checkbox id="all-day" bind:checked={isAllDay} />
					<Label for="all-day">Toute la journee</Label>
				</div>

				<!-- Start date/time -->
				<div class="flex flex-col gap-1.5">
					<Label>Debut</Label>
					<div class="flex gap-2">
						<Popover.Root bind:open={startPickerOpen}>
							<Popover.Trigger class="flex-1">
								<Button
									variant="outline"
									class="w-full cursor-pointer justify-start gap-2 text-left font-normal"
								>
									<iconify-icon icon="solar:calendar-linear" width="16" class="text-muted-foreground shrink-0"></iconify-icon>
									<span class="truncate">{formatDisplayDate(startDateStr)}</span>
								</Button>
							</Popover.Trigger>
							<Popover.Content class="w-auto p-0" align="start">
								<Calendar
									type="single"
									value={startDateVal}
									locale="fr-FR"
									captionLayout="dropdown"
									onValueChange={(v: import('@internationalized/date').DateValue | undefined) => {
										if (v) { startDateStr = v.toString(); startPickerOpen = false; }
									}}
								/>
							</Popover.Content>
						</Popover.Root>
						{#if !isAllDay}
							<Input
								type="time"
								bind:value={startTimeStr}
								class="w-28 cursor-pointer"
							/>
						{/if}
					</div>
				</div>

				<!-- End date/time -->
				<div class="flex flex-col gap-1.5">
					<Label>Fin</Label>
					<div class="flex gap-2">
						<Popover.Root bind:open={endPickerOpen}>
							<Popover.Trigger class="flex-1">
								<Button
									variant="outline"
									class="w-full cursor-pointer justify-start gap-2 text-left font-normal"
								>
									<iconify-icon icon="solar:calendar-linear" width="16" class="text-muted-foreground shrink-0"></iconify-icon>
									<span class="truncate">{formatDisplayDate(endDateStr)}</span>
								</Button>
							</Popover.Trigger>
							<Popover.Content class="w-auto p-0" align="start">
								<Calendar
									type="single"
									value={endDateVal}
									locale="fr-FR"
									captionLayout="dropdown"
									onValueChange={(v: import('@internationalized/date').DateValue | undefined) => {
										if (v) { endDateStr = v.toString(); endPickerOpen = false; }
									}}
								/>
							</Popover.Content>
						</Popover.Root>
						{#if !isAllDay}
							<Input
								type="time"
								bind:value={endTimeStr}
								class="w-28 cursor-pointer"
							/>
						{/if}
					</div>
				</div>

				<!-- Location -->
				<div class="flex flex-col gap-1.5">
					<Label for="location">
						<span class="flex items-center gap-1.5">
							<iconify-icon icon="solar:map-point-linear" width="14" class="text-muted-foreground"></iconify-icon>
							Lieu
						</span>
					</Label>
					<Input
						id="location"
						type="text"
						placeholder="Lieu (optionnel)"
						bind:value={location}
					/>
				</div>

				{#if echoAvailable || conferenceUrl}
					<div class="flex flex-col gap-1.5">
						<Label>
							<span class="flex items-center gap-1.5">
								<iconify-icon icon="solar:videocamera-record-bold-duotone" width="14" class="text-muted-foreground"></iconify-icon>
								Visioconference
							</span>
						</Label>
						{#if conferenceUrl}
							<div class="flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-3 py-2">
								<iconify-icon icon="solar:videocamera-record-bold-duotone" width="18" class="shrink-0 text-primary"></iconify-icon>
								<a href={conferenceUrl} target="_blank" rel="noopener" class="min-w-0 flex-1 truncate text-sm text-primary underline underline-offset-2">
									{conferenceUrl}
								</a>
								<button
									type="button"
									onclick={toggleConference}
									class="flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-full text-muted-foreground hover:bg-accent hover:text-foreground"
									title="Retirer la visioconference"
								>
									<iconify-icon icon="solar:close-circle-linear" width="16"></iconify-icon>
								</button>
							</div>
						{:else}
							<Button variant="outline" onclick={toggleConference} class="w-full cursor-pointer justify-start gap-2">
								<iconify-icon icon="solar:videocamera-record-bold-duotone" width="16"></iconify-icon>
								Ajouter une visio Echo
							</Button>
						{/if}
					</div>
				{/if}

				<!-- Description -->
				<div class="flex flex-col gap-1.5">
					<Label for="description">Description</Label>
					<textarea
						id="description"
						class="border-input dark:bg-input/30 focus-visible:border-ring focus-visible:ring-ring/50 min-h-20 w-full rounded-lg border bg-transparent px-2.5 py-1.5 text-sm outline-none transition-colors focus-visible:ring-3 disabled:opacity-50"
						placeholder="Description (optionnelle)"
						bind:value={description}
						rows="3"
					></textarea>
				</div>

				<!-- Status -->
				<div class="flex flex-col gap-1.5">
					<Label>Statut</Label>
					<Select.Root
						type="single"
						value={status}
						onValueChange={(v: string) => { status = v; }}
					>
						<Select.Trigger class="w-full">
							{status === 'confirmed' ? 'Confirme' : status === 'tentative' ? 'Provisoire' : 'Annule'}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value="confirmed">Confirme</Select.Item>
							<Select.Item value="tentative">Provisoire</Select.Item>
							<Select.Item value="cancelled">Annule</Select.Item>
						</Select.Content>
					</Select.Root>
				</div>

				<!-- Created by (useful for shared calendars) -->
				{#if event?.created_by}
					{@const creator = event.created_by}
					<div class="flex items-center gap-2 border-t border-border pt-3 text-sm text-muted-foreground">
						{#if creator.avatar_url}
							<img
								src={resolveFileUrl(creator.avatar_url)}
								alt={creator.name || creator.email}
								class="size-6 shrink-0 rounded-full border border-border object-cover"
							/>
						{:else}
							<span class="flex size-6 shrink-0 items-center justify-center rounded-full border border-border bg-muted text-[10px] font-semibold">
								{(creator.name || creator.email || '?').trim().slice(0, 2).toUpperCase()}
							</span>
						{/if}
						<span>Cree par <span class="font-medium text-foreground">{creator.name || creator.email}</span></span>
					</div>
				{/if}

				<!-- Error -->
				{#if error}
					<p class="text-sm text-destructive">{error}</p>
				{/if}
			</div>

			<!-- Footer -->
			<div class="mt-6 flex items-center justify-between gap-2">
				<div>
					{#if event}
						<Button
							variant="destructive"
							onclick={handleDelete}
							disabled={deleting || saving}
							class="gap-2"
						>
							<iconify-icon icon="solar:trash-bin-2-linear" width="16"></iconify-icon>
							{deleting ? 'Suppression…' : 'Supprimer'}
						</Button>
					{/if}
				</div>
				<div class="flex gap-2">
					<Button variant="outline" onclick={onClose} disabled={saving || deleting} class="cursor-pointer gap-2">
						Annuler
					</Button>
					<Button onclick={handleSave} disabled={saving || deleting} class="cursor-pointer gap-2">
						<iconify-icon icon="solar:check-circle-linear" width="16"></iconify-icon>
						{saving ? 'Enregistrement…' : 'Enregistrer'}
					</Button>
				</div>
			</div>
		</DialogPrimitive.Content>
	</DialogPrimitive.Portal>
</DialogPrimitive.Root>
