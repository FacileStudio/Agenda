<script lang="ts">
	import { tick } from 'svelte';
	import { scale } from 'svelte/transition';
	import { Cursor, Roll, Stage, Window, cursorAt, type Act } from '@facile/mise-en-scene';
	import WeekGrid from './WeekGrid.svelte';
	import { icons } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
	import { capitalize, formatDate, weekdayName } from '$lib/i18n/format';

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

<Stage
	width={960}
	height={532}
	compact={{ below: 640, width: 358, height: 524 }}
	{script}
	{reset}
	{poster}
	label={t('landing.demo.label')}
>
	{#snippet children({ compact })}
		{#if compact}
			<Window style="top:12px;left:12px;width:334px;height:500px" trailing={status}>
				<div class="wd-body">
					{@render header(true)}
					<WeekGrid days={days.slice(1, 4)} {added} {title} complete={title === eventTitle} compact bind:slot />
				</div>
			</Window>
		{:else}
			<Window style="top:16px;left:40px;width:880px;height:500px" trailing={status}>
				<div class="wd-body">
					{@render header(false)}
					<WeekGrid {days} {added} {title} complete={title === eventTitle} compact={false} bind:slot />
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
