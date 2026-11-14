<script module lang="ts">
	export type WeekDay = { index: number; name: string; number: number };
</script>

<script lang="ts">
	import { scale } from 'svelte/transition';
	import { Icon } from '@facile/muse';
	import { Bar } from '@facile/mise-en-scene';
	import { formatTime } from '$lib/i18n/format';
	import { LANDING_ICONS } from '$lib/landing-icons';

	let {
		days,
		lit,
		added,
		title,
		deadline,
		row
	}: {
		days: WeekDay[];
		lit: boolean;
		added: boolean;
		title: string;
		deadline: string;
		row: number;
	} = $props();

	const FIRST_HOUR = 9;
	const HOURS = Array.from({ length: 8 }, (_, i) => FIRST_HOUR + i);
	const TARGET_DAY = 3;
	const DEADLINE_DAY = 4;
	const TARGET_HOUR = 14;

	const BUSY = [
		{ day: 0, hour: 9.5, length: 1, tone: 1, w: '70%' },
		{ day: 0, hour: 13, length: 2, tone: 3, w: '55%' },
		{ day: 1, hour: 10, length: 1.5, tone: 2, w: '62%' },
		{ day: 2, hour: 9, length: 1, tone: 1, w: '48%' },
		{ day: 2, hour: 11, length: 1, tone: 4, w: '66%' },
		{ day: 3, hour: 12, length: 1, tone: 2, w: '58%' },
		{ day: 3, hour: 16, length: 1, tone: 3, w: '44%' },
		{ day: 4, hour: 10, length: 2, tone: 4, w: '60%' }
	];

	const hourLabel = (hour: number) => formatTime(new Date(2000, 0, 1, hour));
	const top = (hour: number) => (hour - FIRST_HOUR) * row;
</script>

<div class="wg" style:--wg-cols={days.length}>
	<div class="wg-days">
		<span></span>
		{#each days as day (day.index)}
			<span class="wg-day" class:wg-today={day.index === TARGET_DAY}>{day.name}<b>{day.number}</b></span>
		{/each}
	</div>
	<div class="wg-days wg-allday">
		<span></span>
		{#each days as day (day.index)}
			<span class="wg-cell">
				{#if day.index === DEADLINE_DAY}
					<span class="wg-deadline"><Icon icon={LANDING_ICONS.opus} size={13} /><span class="wg-clip">{deadline}</span></span>
				{/if}
			</span>
		{/each}
	</div>
	<div class="wg-grid" style:height="{HOURS.length * row}px">
		<div class="wg-hours">
			{#each HOURS as hour (hour)}
				<span style:top="{top(hour)}px">{hourLabel(hour)}</span>
			{/each}
		</div>
		{#each days as day (day.index)}
			<div class="wg-col">
				{#each HOURS as hour (hour)}
					<span class="wg-slot" class:wg-odd={hour % 2 === 1} style:top="{top(hour)}px" style:height="{row}px"></span>
				{/each}
				{#each BUSY.filter((busy) => busy.day === day.index) as busy (busy.hour)}
					<span
						class="wg-event"
						style:top="{top(busy.hour) + 2}px"
						style:height="{busy.length * row - 4}px"
						style:--wg-tone="var(--color-fc-chart-{busy.tone})"
					>
						<Bar w={busy.w} h={7} />
						<Bar w="32%" h={6} />
					</span>
				{/each}
				{#if day.index === TARGET_DAY}
					<span class="wg-lit" class:wg-on={lit && !added} style:top="{top(TARGET_HOUR) + 2}px" style:height="{row - 4}px"></span>
					{#if added}
						<span
							class="wg-added"
							style:top="{top(TARGET_HOUR) + 2}px"
							style:height="{row - 4}px"
							transition:scale={{ start: 0.85, duration: 260 }}
						>
							<span class="wg-title" class:wg-wrap={row < 40}>{title}</span>
							{#if row >= 40}<span class="wg-time">{hourLabel(TARGET_HOUR)}</span>{/if}
						</span>
					{/if}
				{/if}
			</div>
		{/each}
	</div>
</div>

<style>
	.wg {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.wg-days,
	.wg-grid {
		display: grid;
		grid-template-columns: 44px repeat(var(--wg-cols), minmax(0, 1fr));
	}
	.wg-days {
		font-size: 12px;
		color: var(--color-fc-fg-muted);
	}
	.wg-day {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 4px;
		white-space: nowrap;
	}
	.wg-day b {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		border-radius: 999px;
		font-weight: 700;
	}
	.wg-today {
		color: var(--color-fc-fg);
		font-weight: 700;
	}
	.wg-today b {
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
	}
	.wg-hours {
		position: relative;
		font-size: 11px;
		color: var(--color-fc-fg-muted);
	}
	.wg-hours span:not(:first-child) {
		transform: translateY(-50%);
	}
	.wg-hours span {
		position: absolute;
		right: 6px;
		white-space: nowrap;
	}
	.wg-col {
		position: relative;
		margin: 0 2px;
		overflow: hidden;
		border-radius: 10px;
		background: var(--color-fc-surface);
	}
	.wg-slot {
		position: absolute;
		left: 0;
		right: 0;
	}
	.wg-odd {
		background: color-mix(in oklch, var(--color-fc-field) 50%, transparent);
	}
	.wg-event,
	.wg-added {
		position: absolute;
		left: 3px;
		right: 3px;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		border-radius: 7px;
	}
	.wg-event {
		gap: 5px;
		padding: 7px 8px;
		background: color-mix(in oklch, var(--wg-tone) 30%, var(--color-fc-component));
	}
	.wg-added {
		gap: 2px;
		padding: 4px 7px;
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
		font-size: 12px;
		line-height: 1.2;
		transform-origin: top center;
	}
	.wg-title {
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.wg-time {
		opacity: 0.85;
	}
	.wg-wrap {
		font-size: 11px;
		line-height: 1.1;
		white-space: normal;
	}
	.wg-clip {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.wg-allday {
		height: 26px;
	}
	.wg-cell {
		display: flex;
		min-width: 0;
		padding: 0 5px;
	}
	.wg-deadline {
		display: flex;
		flex: 1;
		align-items: center;
		gap: 5px;
		min-width: 0;
		padding: 0 8px;
		border-radius: 7px;
		background: color-mix(in oklch, var(--color-fc-chart-2) 30%, var(--color-fc-component));
		font-size: 11px;
		font-weight: 700;
		white-space: nowrap;
	}
	.wg-lit {
		position: absolute;
		left: 3px;
		right: 3px;
		border-radius: 7px;
		background: color-mix(in oklch, var(--color-fc-accent) 22%, transparent);
		opacity: 0;
		transition: opacity 300ms ease;
	}
	.wg-on {
		opacity: 1;
	}
</style>
