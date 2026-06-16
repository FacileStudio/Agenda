<script lang="ts">
	import type { AgendaEvent, CalendarItem } from '$lib/backend';

	let {
		events,
		calendars,
		onEventClick
	}: {
		events: AgendaEvent[];
		calendars: CalendarItem[];
		onEventClick: (event: AgendaEvent) => void;
	} = $props();

	function getCalendarColor(calendarId: number): string {
		return calendars.find((c) => c.id === calendarId)?.color ?? '#6b7280';
	}

	function formatTime(dateStr: string): string {
		const d = new Date(dateStr);
		return d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
	}

	function formatDate(dateStr: string): string {
		const d = new Date(dateStr);
		return d.toLocaleDateString('fr-FR', {
			weekday: 'long',
			day: 'numeric',
			month: 'long',
			year: 'numeric'
		});
	}

	function dayKey(dateStr: string): string {
		return dateStr.slice(0, 10);
	}

	const grouped = $derived(() => {
		const sorted = [...events].sort(
			(a, b) => new Date(a.start_at).getTime() - new Date(b.start_at).getTime()
		);
		const map = new Map<string, AgendaEvent[]>();
		for (const ev of sorted) {
			const key = dayKey(ev.start_at);
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(ev);
		}
		return Array.from(map.entries()).map(([key, evs]) => ({
			key,
			date: evs[0].start_at,
			events: evs
		}));
	});
</script>

<div class="flex h-full flex-col overflow-y-auto">
	{#if grouped().length === 0}
		<div class="flex h-full items-center justify-center text-sm text-muted-foreground">
			Aucun événement à afficher
		</div>
	{:else}
		{#each grouped() as group (group.key)}
			<div class="border-b last:border-b-0">
				<div class="sticky top-0 z-10 border-b bg-background px-4 py-2">
					<span class="text-sm font-medium capitalize text-foreground">
						{formatDate(group.date)}
					</span>
				</div>
				<div class="divide-y">
					{#each group.events as event (event.id)}
						<button
							class="flex w-full cursor-pointer items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-accent"
							onclick={() => onEventClick(event)}
						>
							<span
								class="mt-1 size-2.5 flex-shrink-0 rounded-full"
								style="background-color: {getCalendarColor(event.calendar_id)}"
							></span>
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm font-medium">{event.title}</p>
								<p class="text-xs text-muted-foreground">
									{#if event.is_all_day}
										Toute la journée
									{:else}
										{formatTime(event.start_at)} – {formatTime(event.end_at)}
									{/if}
									{#if event.location}
										· {event.location}
									{/if}
								</p>
							</div>
						</button>
					{/each}
				</div>
			</div>
		{/each}
	{/if}
</div>
