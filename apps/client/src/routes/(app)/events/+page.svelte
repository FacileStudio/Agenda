<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend, type AgendaEvent, type CalendarItem } from '$lib/backend';

	let events = $state<(AgendaEvent & { calendar_name?: string })[]>([]);
	let loading = $state(true);

	onMount(async () => {
		try {
			const calendars: CalendarItem[] = await backend.listCalendars();
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
			<div class="flex items-center justify-center py-12 text-muted-foreground">
				<iconify-icon icon="solar:refresh-linear" width="20" class="animate-spin"></iconify-icon>
			</div>
		{:else if events.length === 0}
			<div class="flex flex-col items-center justify-center py-16 text-center">
				<iconify-icon icon="solar:calendar-mark-bold-duotone" width="48" class="text-muted-foreground/50"></iconify-icon>
				<p class="mt-4 text-sm text-muted-foreground">Aucun événement à venir</p>
			</div>
		{:else}
			<div class="mx-auto max-w-2xl space-y-2">
				{#each events as event (event.id)}
					<button
						onclick={() => goto(`/calendar?event=${event.id}`)}
						class="flex w-full items-center gap-4 rounded-lg border border-border p-4 text-left transition-colors hover:bg-muted/50"
					>
						<div class="flex h-10 w-10 shrink-0 flex-col items-center justify-center rounded-lg bg-primary/10">
							<iconify-icon icon="solar:calendar-mark-bold-duotone" width="20" class="text-primary"></iconify-icon>
						</div>
						<div class="min-w-0 flex-1">
							<p class="truncate text-sm font-medium">{event.title || 'Sans titre'}</p>
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
						<iconify-icon icon="solar:alt-arrow-right-linear" width="16" class="shrink-0 text-muted-foreground"></iconify-icon>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
