<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import * as Select from '$lib/components/ui/select';
	import { cn } from '$lib/utils';
	import type { AgendaEvent, CalendarItem, CreateEventRequest } from '$lib/backend';

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

	// Derive default values from event or initialDate
	function defaultStart(): string {
		if (event) return new Date(event.start_at).toISOString().slice(0, 16);
		if (initialDate) return new Date(initialDate + 'T09:00').toISOString().slice(0, 16);
		const d = new Date();
		d.setMinutes(0, 0, 0);
		return d.toISOString().slice(0, 16);
	}

	function defaultEnd(): string {
		if (event) return new Date(event.end_at).toISOString().slice(0, 16);
		if (initialDate) return new Date(initialDate + 'T10:00').toISOString().slice(0, 16);
		const d = new Date();
		d.setMinutes(0, 0, 0);
		d.setHours(d.getHours() + 1);
		return d.toISOString().slice(0, 16);
	}

	let title = $state(event?.title ?? '');
	let description = $state(event?.description ?? '');
	let location = $state(event?.location ?? '');
	let startAt = $state(defaultStart());
	let endAt = $state(defaultEnd());
	let isAllDay = $state(event?.is_all_day ?? false);
	let status = $state(event?.status ?? 'confirmed');
	let calendarId = $state<number>(
		event?.calendar_id ?? (calendars[0]?.id ?? 0)
	);
	let saving = $state(false);
	let deleting = $state(false);
	let error = $state('');

	// Reset form when event/initialDate changes
	$effect(() => {
		if (open) {
			title = event?.title ?? '';
			description = event?.description ?? '';
			location = event?.location ?? '';
			startAt = defaultStart();
			endAt = defaultEnd();
			isAllDay = event?.is_all_day ?? false;
			status = event?.status ?? 'confirmed';
			calendarId = event?.calendar_id ?? (calendars[0]?.id ?? 0);
			error = '';
		}
	});

	async function handleSave() {
		if (!title.trim()) {
			error = 'Le titre est requis.';
			return;
		}
		if (!calendarId) {
			error = 'Sélectionnez un calendrier.';
			return;
		}
		saving = true;
		error = '';
		try {
			const data: CreateEventRequest = {
				title: title.trim(),
				description: description || undefined,
				location: location || undefined,
				start_at: new Date(startAt).toISOString(),
				end_at: new Date(endAt).toISOString(),
				is_all_day: isAllDay,
				status: status || undefined
			};
			await onSave(calendarId, data);
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Une erreur est survenue.';
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!confirm('Supprimer cet événement ?')) return;
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
					{event ? 'Modifier l\'événement' : 'Nouvel événement'}
				</h2>
				<DialogPrimitive.Close
					class="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="size-4"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<line x1="18" y1="6" x2="6" y2="18" />
						<line x1="6" y1="6" x2="18" y2="18" />
					</svg>
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
						placeholder="Titre de l'événement"
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
							{calendars.find((c) => c.id === calendarId)?.name ?? 'Choisir un calendrier'}
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
					<Label for="all-day">Toute la journée</Label>
				</div>

				<!-- Start / End dates -->
				<div class="grid grid-cols-2 gap-3">
					<div class="flex flex-col gap-1.5">
						<Label for="start-at">Début</Label>
						<Input
							id="start-at"
							type={isAllDay ? 'date' : 'datetime-local'}
							value={isAllDay ? startAt.slice(0, 10) : startAt}
							oninput={(e) => {
								const v = (e.target as HTMLInputElement).value;
								startAt = isAllDay ? v + 'T00:00' : v;
							}}
						/>
					</div>
					<div class="flex flex-col gap-1.5">
						<Label for="end-at">Fin</Label>
						<Input
							id="end-at"
							type={isAllDay ? 'date' : 'datetime-local'}
							value={isAllDay ? endAt.slice(0, 10) : endAt}
							oninput={(e) => {
								const v = (e.target as HTMLInputElement).value;
								endAt = isAllDay ? v + 'T23:59' : v;
							}}
						/>
					</div>
				</div>

				<!-- Location -->
				<div class="flex flex-col gap-1.5">
					<Label for="location">Lieu</Label>
					<Input
						id="location"
						type="text"
						placeholder="Lieu (optionnel)"
						bind:value={location}
					/>
				</div>

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
							{status === 'confirmed' ? 'Confirmé' : status === 'tentative' ? 'Provisoire' : 'Annulé'}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value="confirmed">Confirmé</Select.Item>
							<Select.Item value="tentative">Provisoire</Select.Item>
							<Select.Item value="cancelled">Annulé</Select.Item>
						</Select.Content>
					</Select.Root>
				</div>

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
						>
							{deleting ? 'Suppression…' : 'Supprimer'}
						</Button>
					{/if}
				</div>
				<div class="flex gap-2">
					<Button variant="outline" onclick={onClose} disabled={saving || deleting}>
						Annuler
					</Button>
					<Button onclick={handleSave} disabled={saving || deleting}>
						{saving ? 'Enregistrement…' : 'Enregistrer'}
					</Button>
				</div>
			</div>
		</DialogPrimitive.Content>
	</DialogPrimitive.Portal>
</DialogPrimitive.Root>
