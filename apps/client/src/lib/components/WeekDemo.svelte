<script lang="ts">
	import { scale } from 'svelte/transition';
	import { Roll, Stage, Window, type Act } from '@facile/mise-en-scene';
	import { Icon, icons } from '@facile/muse';
	import BulleAsk from './BulleAsk.svelte';
	import WeekGrid from './WeekGrid.svelte';
	import { t } from '$lib/i18n/index.svelte';
	import { capitalize, formatDate, weekdayName } from '$lib/i18n/format';
	import { LANDING_ICONS } from '$lib/landing-icons';

	const VIEWS = [LANDING_ICONS.day, icons.calendar, LANDING_ICONS.week, LANDING_ICONS.list];

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
	const prompt = $derived(t('landing.demo.prompt'));
	const replyWords = $derived(t('landing.demo.reply').split(' ').length);

	let draft = $state('');
	let sent = $state(false);
	let lit = $state(false);
	let added = $state(false);
	let words = $state(0);

	function reset() {
		draft = '';
		sent = false;
		lit = false;
		added = false;
		words = 0;
	}

	function poster() {
		sent = true;
		added = true;
		words = replyWords;
	}

	async function script(act: Act) {
		await act.wait(900);
		await act.type(prompt, (value) => (draft = value), 55);
		await act.wait(400);
		draft = '';
		sent = true;
		await act.wait(700);
		lit = true;
		await act.wait(900);
		await act.count(replyWords, (value) => (words = value), 90);
		await act.wait(300);
		added = true;
		await act.wait(3000);
		reset();
		await act.wait(700);
	}
</script>

{#snippet status()}
	{#if added}
		<span class="wd-status" transition:scale={{ start: 0.9, duration: 220 }}>
			<Icon icon={LANDING_ICONS.synced} size={14} />
			<Roll value={t('landing.demo.synced')} />
		</span>
	{/if}
{/snippet}

{#snippet header(compact: boolean)}
	<div class="wd-head">
		<span class="wd-month">{month}</span>
		<span class="wd-views">
			{#each compact ? VIEWS.slice(0, 2) : VIEWS as view, i (view)}
				<span class="wd-view" class:wd-active={i === 1}><Icon icon={view} size={16} /></span>
			{/each}
		</span>
		<span class="wd-new">
			<Icon icon={icons.plus} size={16} />
			{#if !compact}{t('calendar.new')}{/if}
		</span>
	</div>
{/snippet}

<Stage
	width={960}
	height={600}
	compact={{ below: 640, width: 358, height: 660 }}
	{script}
	{reset}
	{poster}
	label={t('landing.demo.label')}
>
	{#snippet children({ compact })}
		{#if compact}
			<Window style="top:12px;left:12px;width:334px;height:460px" trailing={status}>
				<div class="wd-body">
					{@render header(true)}
					<WeekGrid days={days.slice(2, 5)} {lit} {added} title={t('landing.demo.event')} deadline={t('landing.demo.deadline')} row={34} />
				</div>
			</Window>
			<BulleAsk {draft} {sent} {words} style="top:430px;left:24px;width:310px;height:196px" />
		{:else}
			<Window style="top:16px;left:160px;width:780px;height:540px" trailing={status}>
				<div class="wd-body">
					{@render header(false)}
					<WeekGrid {days} {lit} {added} title={t('landing.demo.event')} deadline={t('landing.demo.deadline')} row={44} />
				</div>
			</Window>
			<BulleAsk {draft} {sent} {words} style="top:300px;left:20px;width:300px;height:204px" />
		{/if}
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
		color: var(--color-fc-success);
		font-size: 11px;
		font-weight: 700;
		white-space: nowrap;
	}
</style>
