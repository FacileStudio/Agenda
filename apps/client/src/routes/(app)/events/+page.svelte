<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { EmptyState, Spinner, icons } from '@facile/muse';
	import { backend, type AgendaEvent, type CalendarItem } from '$lib/backend';
	import { spaceId } from '$lib/space-context.svelte';

	let events = $state<(AgendaEvent & { calendar_name?: string })[]>([]);
	let loading = $state(true);

	onMount(async () => {
		try {
			const calendars: CalendarItem[] = await backend.listCalendars(spaceId() ?? undefined);
			const now = new Date();
			const from = now.toISOString();
			const to = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000).toISOString();

			const allEvents: (AgendaEvent & { calendar_name?: string })[] = [];
			for (const cal of calendars) {
				try {
					const calEvents = await backend.listEvents(cal.id, from, to);
					const arr = Array.isArray(calEvents) ? calEvents : [];
					for (const ev of arr) {
						allEvents.push({ ...ev, calendar_name: cal.name });
					}
				} catch {}
			}
			allEvents.sort((a, b) => new Date(a.start_at).getTime() - new Date(b.start_at).getTime());
			events = allEvents;
		} catch {
			events = [];
		}
		loading = false;
	});

	function formatDate(iso: string) {
		const d = new Date(iso);
		return d.toLocaleDateString('fr-FR', { weekday: 'short', day: 'numeric', month: 'short' });
	}

	function formatTime(iso: string) {
		const d = new Date(iso);
		return d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
	}
</script>

<svelte:head>
	<title>Événements — Agenda</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<h1 class="text-lg font-semibold">Événements</h1>
		<p class="mt-0.5 text-sm text-muted-foreground">Vos événements des 30 prochains jours</p>
	</div>

	<div class="flex-1 overflow-auto p-4 md:p-8">
		{#if loading}
			<div class="flex items-center justify-center py-12">
				<Spinner label="Chargement" />
			</div>
		{:else if events.length === 0}
			<div class="mx-auto max-w-2xl">
				<EmptyState
					icon={icons.calendar}
					title="Aucun événement à venir"
					description="Vos rendez-vous des 30 prochains jours apparaîtront ici."
				/>
			</div>
		{:else}
			<div class="mx-auto flex max-w-2xl flex-col gap-2">
				{#each events as event (event.id)}
					<button
						type="button"
						onclick={() => goto(`/calendar?event=${event.id}`)}
						class="flex w-full cursor-pointer items-center gap-4 rounded-fc-lg bg-card p-4 text-left transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
					>
						<div class="flex size-10 shrink-0 items-center justify-center rounded-fc-md bg-muted text-muted-foreground">
							<iconify-icon icon={icons.calendar} width="20" height="20" class="block size-5"
							></iconify-icon>
						</div>
						<div class="min-w-0 flex-1">
							<p class="truncate text-sm font-medium text-foreground">{event.title || 'Sans titre'}</p>
							<div class="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
								<span>{formatDate(event.start_at)}</span>
								{#if !event.is_all_day}
									<span>{formatTime(event.start_at)} — {formatTime(event.end_at)}</span>
								{:else}
									<span>Toute la journée</span>
								{/if}
								{#if event.calendar_name}
									<span class="truncate">· {event.calendar_name}</span>
								{/if}
							</div>
						</div>
						<iconify-icon
							icon={icons.arrow}
							width="16"
							height="16"
							class="block size-4 shrink-0 text-muted-foreground"
						></iconify-icon>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
