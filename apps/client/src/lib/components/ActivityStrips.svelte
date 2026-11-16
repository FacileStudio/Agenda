<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Tooltip } from '@facile/muse';
	import { LANE_WIDTH, dayStrips, spanLabel, startOfDay, type Strip } from '$lib/activity-lanes';
	import type { Activity } from '$lib/backend';
	import { formatTime } from '$lib/i18n/format';
	import { t } from '$lib/i18n/index.svelte';

	let { activities, day, slotHeight }: { activities: Activity[]; day: Date; slotHeight: number } = $props();

	const MIN_HEIGHT = 10;
	const TONE = { sablier: 'bg-fc-chart-2', echo: 'bg-fc-chart-3' } as const;
	const NAME = { sablier: 'Sablier', echo: 'Echo' } as const;
	const HIT = 'group absolute z-20 flex justify-end rounded-full focus-visible:outline-2 focus-visible:outline-ring';
	const BAR = 'h-full w-1.5 rounded-full transition-[width,filter] group-hover:w-2 group-hover:brightness-110 group-focus-visible:w-2 motion-reduce:transition-none';

	let now = $state(Date.now());
	const tick = setInterval(() => (now = Date.now()), 60000);
	onDestroy(() => clearInterval(tick));

	const dayStart = $derived(startOfDay(day));

	const strips = $derived(
		dayStrips(activities, dayStart, now).map((strip) => ({
			...strip,
			top: ((strip.from - dayStart) / 3600000) * slotHeight,
			height: Math.max(((strip.to - strip.from) / 3600000) * slotHeight, MIN_HEIGHT),
			label: describe(strip)
		}))
	);

	function describe(strip: Strip): string {
		const { activity } = strip;
		const start = formatTime(activity.start_at);
		const when = activity.end_at ? `${start} – ${formatTime(activity.end_at)}` : t('activity.running', { time: start });
		const kind = t(activity.source === 'echo' ? 'activity.echo' : 'activity.sablier');
		const title = activity.title ? `${NAME[activity.source]} · ${activity.title}` : kind;
		return `${title} · ${when} · ${spanLabel(strip.to - strip.from)}`;
	}

	function place(strip: { lane: number; top: number; height: number }): string {
		return `top: ${strip.top}px; height: ${strip.height}px; right: ${strip.lane * LANE_WIDTH}px; width: ${LANE_WIDTH}px;`;
	}
</script>

{#each strips as strip (strip.activity.source + strip.activity.id)}
	<Tooltip text={strip.label} describe={false} delay={80} class="contents">
		{#if strip.activity.url}
			<a
				href={strip.activity.url}
				target="_blank"
				rel="noopener"
				aria-label={strip.label}
				class="{HIT} cursor-pointer"
				style={place(strip)}
				onpointerdown={(e) => e.stopPropagation()}
			>
				<span class="{BAR} {TONE[strip.activity.source]}"></span>
			</a>
		{:else}
			<span role="img" aria-label={strip.label} class={HIT} style={place(strip)}>
				<span class="{BAR} {TONE[strip.activity.source]}"></span>
			</span>
		{/if}
	</Tooltip>
{/each}
