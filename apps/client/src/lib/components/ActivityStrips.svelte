<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Tooltip } from '@facile/muse';
	import type { Activity } from '$lib/backend';
	import { formatTime } from '$lib/i18n/format';
	import { t } from '$lib/i18n/index.svelte';

	let { activities, day, slotHeight }: { activities: Activity[]; day: Date; slotHeight: number } = $props();

	const DAY_MS = 86400000;
	const MIN_HEIGHT = 6;
	const LANE = { sablier: 'right-0 bg-fc-chart-2', echo: 'right-1 bg-fc-chart-3' } as const;

	let now = $state(Date.now());
	const tick = setInterval(() => (now = Date.now()), 60000);
	onDestroy(() => clearInterval(tick));

	const dayStart = $derived(new Date(day.getFullYear(), day.getMonth(), day.getDate()).getTime());

	const strips = $derived(
		activities.flatMap((activity) => {
			const start = new Date(activity.start_at).getTime();
			const end = activity.end_at ? new Date(activity.end_at).getTime() : Math.max(now, start);
			const from = Math.max(start, dayStart);
			const to = Math.min(end, dayStart + DAY_MS);
			if (to <= from) return [];
			const top = ((from - dayStart) / 3600000) * slotHeight;
			const height = Math.max(((to - from) / 3600000) * slotHeight, MIN_HEIGHT);
			return [{ activity, top, height, label: describe(activity) }];
		})
	);

	function describe(activity: Activity): string {
		const title = activity.title || t(activity.source === 'echo' ? 'activity.echo' : 'activity.sablier');
		const start = formatTime(activity.start_at);
		const when = activity.end_at ? `${start} – ${formatTime(activity.end_at)}` : t('activity.running', { time: start });
		return `${title} · ${when}`;
	}
</script>

{#each strips as strip (strip.activity.source + strip.activity.id)}
	<Tooltip text={strip.label} describe={false} class="contents">
		{#if strip.activity.url}
			<a
				href={strip.activity.url}
				target="_blank"
				rel="noopener"
				aria-label={strip.label}
				class="absolute z-20 w-[3px] rounded-full focus-visible:outline-2 focus-visible:outline-ring {LANE[strip.activity.source]}"
				style="top: {strip.top}px; height: {strip.height}px;"
				onpointerdown={(e) => e.stopPropagation()}
			></a>
		{:else}
			<span
				role="img"
				aria-label={strip.label}
				class="absolute z-20 w-[3px] rounded-full {LANE[strip.activity.source]}"
				style="top: {strip.top}px; height: {strip.height}px;"
			></span>
		{/if}
	</Tooltip>
{/each}
