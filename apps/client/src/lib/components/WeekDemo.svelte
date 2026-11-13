<script lang="ts">
	import { tick } from 'svelte';
	import { scale } from 'svelte/transition';
	import { Bar, Cursor, Roll, Stage, Window, cursorAt, type Act } from '@facile/mise-en-scene';
	import { icons } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
	import { capitalize, formatDate, formatTime, weekdayName } from '$lib/i18n/format';

	const FIRST_HOUR = 9;
	const HOURS = Array.from({ length: 8 }, (_, i) => FIRST_HOUR + i);
	const ROW = 44;
	const TARGET_DAY = 2;
	const TARGET_HOUR = 14;

	const BUSY = [
		{ day: 0, hour: 9.5, length: 1, tone: 1, w: '70%' },
		{ day: 0, hour: 13, length: 2, tone: 3, w: '55%' },
		{ day: 1, hour: 10, length: 1.5, tone: 2, w: '62%' },
		{ day: 2, hour: 9, length: 1, tone: 1, w: '48%' },
		{ day: 2, hour: 11, length: 1, tone: 4, w: '66%' },
		{ day: 3, hour: 12, length: 1, tone: 2, w: '58%' },
		{ day: 3, hour: 15, length: 1.5, tone: 3, w: '44%' },
		{ day: 4, hour: 10, length: 2, tone: 4, w: '60%' }
	];

	const VIEWS = ['solar:calendar-minimalistic-linear', icons.calendar, 'solar:calendar-date-linear', 'solar:list-linear'];

	const monday = (() => {
		const now = new Date();
		const start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
		start.setDate(start.getDate() - ((start.getDay() + 6) % 7));
		return start;
	})();

	const days = $derived(
		Array.from({ length: 5 }, (_, i) => {
			const date = new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + i);
			return { index: i, name: weekdayName(i, 'short'), number: date.getDate() };
		})
	);
	const month = $derived(capitalize(formatDate(monday, { month: 'long', year: 'numeric' })));
	const hourLabel = (hour: number) => formatTime(new Date(2000, 0, 1, hour));

	let added = $state(false);
	let title = $state('');
	let synced = $state(false);
	let cursor = $state(cursorAt());
	let slot = $state<HTMLElement>();

	const eventTitle = $derived(t('landing.demo.event'));

	function reset() {
		added = false;
		title = '';
		synced = false;
		cursor.visible = false;
	}

	function poster() {
		added = true;
		title = eventTitle;
		synced = true;
	}

	async function script(act: Act) {
		await act.wait(1200);
		cursor.visible = true;
		await act.move(cursor, act.point(slot, 0.55, 0.5));
		await act.click(cursor, () => (added = true));
		await tick();
		await act.wait(250);
		await act.type(eventTitle, (value) => (title = value), 85);
		await act.wait(900);
		cursor.visible = false;
		await act.wait(1400);
		synced = true;
		await act.wait(2800);
		reset();
		await act.wait(600);
	}
</script>

{#snippet status()}
	{#if added}
		<span class="wd-status" class:wd-done={synced} transition:scale={{ start: 0.9, duration: 220 }}>
			<iconify-icon icon={synced ? 'solar:check-circle-bold' : 'solar:refresh-linear'} width="14" height="14" class="block size-3.5"></iconify-icon>
			<Roll value={synced ? t('landing.demo.synced') : t('landing.demo.syncing')} />
		</span>
	{/if}
{/snippet}

{#snippet header(compact: boolean)}
	<div class="wd-head">
		<span class="wd-month">{month}</span>
		<span class="wd-views">
			{#each compact ? VIEWS.slice(0, 2) : VIEWS as view, i (view)}
				<span class="wd-view" class:wd-active={i === 1}>
					<iconify-icon icon={view} width="16" height="16" class="block size-4"></iconify-icon>
				</span>
			{/each}
		</span>
		<span class="wd-new">
			<iconify-icon icon={icons.plus} width="16" height="16" class="block size-4"></iconify-icon>
			{#if !compact}{t('calendar.new')}{/if}
		</span>
	</div>
{/snippet}

{#snippet grid(shown: typeof days)}
	<div class="wd-days" style:--wd-cols={shown.length}>
		<span></span>
		{#each shown as day (day.index)}
			<span class="wd-day" class:wd-today={day.index === TARGET_DAY}>
				{day.name}<b>{day.number}</b>
			</span>
		{/each}
	</div>
	<div class="wd-grid" style:--wd-cols={shown.length} style:height="{HOURS.length * ROW}px">
		<div class="wd-hours">
			{#each HOURS as hour (hour)}
				<span style:top="{(hour - FIRST_HOUR) * ROW}px">{hourLabel(hour)}</span>
			{/each}
		</div>
		{#each shown as day (day.index)}
			<div class="wd-col">
				{#each HOURS as hour (hour)}
					<span class="wd-slot" class:wd-odd={hour % 2 === 1} style:top="{(hour - FIRST_HOUR) * ROW}px" style:height="{ROW}px"></span>
				{/each}
				{#each BUSY.filter((busy) => busy.day === day.index) as busy (busy.hour)}
					<span
						class="wd-event"
						style:top="{(busy.hour - FIRST_HOUR) * ROW + 2}px"
						style:height="{busy.length * ROW - 4}px"
						style:--wd-tone="var(--color-fc-chart-{busy.tone})"
					>
						<Bar w={busy.w} h={7} />
						<Bar w="32%" h={6} />
					</span>
				{/each}
				{#if day.index === TARGET_DAY}
					<span
						bind:this={slot}
						class="wd-target"
						style:top="{(TARGET_HOUR - FIRST_HOUR) * ROW}px"
						style:height="{ROW}px"
					></span>
					{#if added}
						<span
							class="wd-added"
							style:top="{(TARGET_HOUR - FIRST_HOUR) * ROW + 2}px"
							style:height="{ROW - 4}px"
							transition:scale={{ start: 0.85, duration: 260 }}
						>
							<span class="wd-title">{title}<span class="wd-caret" class:wd-hide={title === eventTitle}></span></span>
							<span class="wd-time">{hourLabel(TARGET_HOUR)}</span>
						</span>
					{/if}
				{/if}
			</div>
		{/each}
	</div>
{/snippet}

<Stage
	width={960}
	height={532}
	compact={{ below: 640, width: 358, height: 496 }}
	{script}
	{reset}
	{poster}
	label={t('landing.demo.label')}
>
	{#snippet children({ compact })}
		{#if compact}
			<Window style="top:12px;left:12px;width:334px;height:472px" trailing={status}>
				<div class="wd-body wd-compact">
					{@render header(true)}
					{@render grid(days.slice(1, 4))}
				</div>
			</Window>
		{:else}
			<Window style="top:16px;left:40px;width:880px;height:500px" trailing={status}>
				<div class="wd-body">
					{@render header(false)}
					{@render grid(days)}
				</div>
			</Window>
		{/if}
		<Cursor {cursor} />
	{/snippet}
</Stage>

<style>
	.wd-body {
		display: flex;
		flex: 1;
		flex-direction: column;
		gap: 10px;
		min-height: 0;
		padding: 14px 16px 16px;
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
	}
	.wd-head {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.wd-month {
		flex: 1;
		min-width: 0;
		font-family: var(--font-fc-title);
		font-size: 18px;
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.wd-views {
		display: flex;
		gap: 4px;
		padding: 4px;
		border-radius: 999px;
		background: var(--color-fc-surface);
	}
	.wd-view {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 30px;
		border-radius: 999px;
		color: var(--color-fc-fg-muted);
	}
	.wd-active {
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
	}
	.wd-new {
		display: flex;
		align-items: center;
		gap: 6px;
		height: 38px;
		padding: 0 14px;
		border-radius: 999px;
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
		font-size: 13px;
		font-weight: 700;
	}
	.wd-days,
	.wd-grid {
		display: grid;
		grid-template-columns: 44px repeat(var(--wd-cols, 5), minmax(0, 1fr));
	}
	.wd-days {
		font-size: 12px;
		color: var(--color-fc-fg-muted);
	}
	.wd-day {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 4px;
		white-space: nowrap;
	}
	.wd-day b {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		border-radius: 999px;
		font-weight: 700;
	}
	.wd-today {
		color: var(--color-fc-fg);
		font-weight: 700;
	}
	.wd-today b {
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
	}
	.wd-hours {
		position: relative;
		font-size: 11px;
		color: var(--color-fc-fg-muted);
	}
	.wd-hours span {
		position: absolute;
		right: 6px;
		transform: translateY(-50%);
		white-space: nowrap;
	}
	.wd-hours span:first-child {
		transform: none;
	}
	.wd-col {
		position: relative;
		margin: 0 2px;
		overflow: hidden;
		border-radius: 10px;
		background: var(--color-fc-surface);
	}
	.wd-slot,
	.wd-target {
		position: absolute;
		left: 0;
		right: 0;
	}
	.wd-odd {
		background: color-mix(in oklch, var(--color-fc-field) 50%, transparent);
	}
	.wd-event,
	.wd-added {
		position: absolute;
		left: 3px;
		right: 3px;
		display: flex;
		flex-direction: column;
		gap: 5px;
		padding: 7px 8px;
		border-radius: 7px;
		overflow: hidden;
	}
	.wd-event {
		background: color-mix(in oklch, var(--wd-tone) 30%, var(--color-fc-component));
	}
	.wd-added {
		gap: 2px;
		padding: 5px 8px;
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
		font-size: 12px;
		line-height: 1.2;
		transform-origin: top center;
	}
	.wd-title {
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.wd-time {
		opacity: 0.85;
	}
	.wd-compact .wd-title {
		white-space: normal;
		overflow-wrap: anywhere;
	}
	.wd-compact .wd-time {
		display: none;
	}
	.wd-caret {
		display: inline-block;
		width: 1.5px;
		height: 12px;
		margin-left: 1px;
		vertical-align: -2px;
		background: currentColor;
	}
	.wd-hide {
		visibility: hidden;
	}
	.wd-status {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		height: 24px;
		padding: 0 10px;
		border-radius: 999px;
		background: var(--color-fc-surface);
		color: var(--color-fc-fg-muted);
		font-size: 11px;
		font-weight: 700;
		white-space: nowrap;
	}
	.wd-done {
		color: var(--color-fc-success);
	}
</style>
