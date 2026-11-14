<script lang="ts">
	import { Icon, icons } from '@facile/muse';
	import { Bar, Stage, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';
	import { capitalize, formatDate } from '$lib/i18n/format';
	import { LANDING_ICONS } from '$lib/landing-icons';

	const friday = (() => {
		const now = new Date();
		const date = new Date(now.getFullYear(), now.getMonth(), now.getDate());
		date.setDate(date.getDate() - ((date.getDay() + 6) % 7) + 4);
		return date;
	})();
	const day = $derived(capitalize(formatDate(friday, { weekday: 'short', day: 'numeric' })));

	let shown = $state(false);

	async function script(act: Act) {
		await act.wait(900);
		shown = true;
		await act.wait(2800);
		shown = false;
		await act.wait(700);
	}
</script>

<Stage width={320} height={200} {script} reset={() => (shown = false)} poster={() => (shown = true)}>
	<div class="os-task">
		<span class="os-app"><Icon icon={LANDING_ICONS.opus} size={14} />Opus</span>
		<b>{t('landing.demo.deadline')}</b>
		<Bar w="80%" h={6} />
		<span class="os-due"><Icon icon={icons.clock} size={12} />{day}</span>
	</div>
	<span class="os-arrow"><Icon icon={icons.arrow} size={20} /></span>
	<div class="os-day">
		<span class="os-head">{day}</span>
		<span class="os-chip" class:os-in={shown}><Icon icon={LANDING_ICONS.opus} size={12} /><span class="os-clip">{t('landing.demo.deadline')}</span></span>
		<span class="os-busy"><Bar w="60%" h={6} /></span>
		<span class="os-busy os-late"><Bar w="45%" h={6} /></span>
	</div>
</Stage>

<style>
	.os-task,
	.os-day {
		position: absolute;
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 12px;
		border-radius: 14px;
		background: var(--color-fc-surface);
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
		font-size: 12px;
	}
	.os-task {
		top: 44px;
		left: 14px;
		width: 128px;
		height: 112px;
	}
	.os-app,
	.os-due,
	.os-chip {
		display: flex;
		align-items: center;
		gap: 5px;
		white-space: nowrap;
	}
	.os-app {
		color: var(--color-fc-fg-muted);
		font-weight: 700;
	}
	.os-due {
		color: var(--color-fc-fg-muted);
	}
	.os-arrow {
		position: absolute;
		top: 90px;
		left: 148px;
		color: var(--color-fc-fg-muted);
	}
	.os-day {
		top: 20px;
		left: 176px;
		width: 130px;
		height: 160px;
	}
	.os-head {
		font-weight: 700;
	}
	.os-chip {
		height: 24px;
		padding: 0 8px;
		overflow: hidden;
		border-radius: 7px;
		background: color-mix(in oklch, var(--color-fc-chart-2) 30%, var(--color-fc-component));
		font-size: 11px;
		font-weight: 700;
		opacity: 0;
		transform: scale(0.85);
		transition:
			opacity 260ms ease,
			transform 260ms ease;
	}
	.os-clip {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.os-in {
		opacity: 1;
		transform: none;
	}
	.os-busy {
		display: flex;
		height: 34px;
		padding: 8px;
		border-radius: 7px;
		background: color-mix(in oklch, var(--color-fc-chart-1) 30%, var(--color-fc-component));
	}
	.os-late {
		background: color-mix(in oklch, var(--color-fc-chart-3) 30%, var(--color-fc-component));
	}
</style>
