<script lang="ts">
	import { Icon, icons } from '@facile/muse';
	import { Stage, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';
	import { capitalize, formatDate, formatTime } from '$lib/i18n/format';

	const PEOPLE = [
		{ initial: 'L', tone: 1 },
		{ initial: 'M', tone: 2 },
		{ initial: 'S', tone: 3 },
		{ initial: 'A', tone: 4 }
	];

	const when = $derived(
		`${capitalize(formatDate(new Date(2026, 0, 6), { weekday: 'short' }))} · ${formatTime(new Date(2026, 0, 6, 10))}`
	);

	let seen = $state(0);

	async function script(act: Act) {
		await act.wait(700);
		await act.count(PEOPLE.length, (value) => (seen = value), 380);
		await act.wait(2600);
		seen = 0;
		await act.wait(600);
	}
</script>

<Stage width={320} height={200} {script} reset={() => (seen = 0)} poster={() => (seen = PEOPLE.length)}>
	<div class="ts-event">
		<b>{t('landing.demo.team')}</b>
		<span class="ts-when"><Icon icon={icons.clock} size={12} />{when}</span>
	</div>
	<div class="ts-people">
		{#each PEOPLE as person, i (person.initial)}
			<span class="ts-person" class:ts-seen={i < seen} style:--ts-tone="var(--color-fc-chart-{person.tone})">
				{person.initial}
				<span class="ts-check"><Icon icon={icons.check} size={12} /></span>
			</span>
		{/each}
	</div>
</Stage>

<style>
	.ts-event {
		position: absolute;
		top: 36px;
		left: 40px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		width: 240px;
		height: 64px;
		padding: 12px 14px;
		border-radius: 12px;
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
		font-family: var(--font-fc-body);
		font-size: 13px;
	}
	.ts-when {
		display: flex;
		align-items: center;
		gap: 5px;
		font-size: 12px;
		opacity: 0.85;
	}
	.ts-people {
		position: absolute;
		top: 128px;
		left: 40px;
		display: flex;
		justify-content: space-between;
		width: 240px;
	}
	.ts-person {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 44px;
		height: 44px;
		border-radius: 999px;
		background: color-mix(in oklch, var(--ts-tone) 40%, var(--color-fc-component));
		color: var(--color-fc-fg);
		font-family: var(--font-fc-title);
		font-weight: 700;
	}
	.ts-check {
		position: absolute;
		right: -3px;
		bottom: -3px;
		display: flex;
		padding: 2px;
		border-radius: 999px;
		background: var(--color-fc-page);
		color: var(--color-fc-success);
		opacity: 0;
		transform: scale(0.6);
		transition:
			opacity 220ms ease,
			transform 220ms ease;
	}
	.ts-seen .ts-check {
		opacity: 1;
		transform: none;
	}
</style>
