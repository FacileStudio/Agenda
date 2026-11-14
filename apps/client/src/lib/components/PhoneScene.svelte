<script lang="ts">
	import { Icon } from '@facile/muse';
	import { Bar, Roll, Stage, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';
	import { formatTime } from '$lib/i18n/format';
	import { LANDING_ICONS } from '$lib/landing-icons';

	const at = (hour: number) => formatTime(new Date(2026, 0, 8, hour));

	let landed = $state(false);

	async function script(act: Act) {
		await act.wait(900);
		landed = true;
		await act.wait(2800);
		landed = false;
		await act.wait(700);
	}
</script>

<Stage width={320} height={200} {script} reset={() => (landed = false)} poster={() => (landed = true)}>
	<div class="ps-laptop">
		<Icon icon={LANDING_ICONS.laptop} size={40} />
		<span class="ps-sync" class:ps-on={landed}>
			<Icon icon={LANDING_ICONS.synced} size={12} />
			<Roll value={t('landing.demo.synced')} />
		</span>
	</div>
	<div class="ps-phone">
		<span class="ps-notch"></span>
		<b class="ps-today">{t('landing.demo.today')}</b>
		<span class="ps-row"><span class="ps-time">{at(10)}</span><Bar w="60%" h={6} /></span>
		<span class="ps-row ps-new" class:ps-on={landed}>
			<span class="ps-time">{at(14)}</span>{t('landing.demo.event')}
		</span>
		<span class="ps-row"><span class="ps-time">{at(16)}</span><Bar w="50%" h={6} /></span>
	</div>
</Stage>

<style>
	.ps-laptop {
		position: absolute;
		top: 56px;
		left: 18px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10px;
		width: 120px;
		color: var(--color-fc-fg-muted);
		font-family: var(--font-fc-body);
	}
	.ps-sync {
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
		opacity: 0;
		transition: opacity 260ms ease;
	}
	.ps-phone {
		position: absolute;
		top: 10px;
		left: 160px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		width: 132px;
		height: 180px;
		padding: 26px 10px 10px;
		border-radius: 26px;
		background: var(--color-fc-surface);
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
		font-size: 11px;
	}
	.ps-notch {
		position: absolute;
		top: 9px;
		left: 50%;
		width: 36px;
		height: 8px;
		border-radius: 999px;
		background: var(--color-fc-field);
		transform: translateX(-50%);
	}
	.ps-today {
		font-family: var(--font-fc-title);
		font-size: 13px;
	}
	.ps-row {
		display: flex;
		align-items: center;
		gap: 6px;
		height: 28px;
		padding: 0 6px;
		overflow: hidden;
		border-radius: 7px;
		background: var(--color-fc-component);
		white-space: nowrap;
	}
	.ps-time {
		color: var(--color-fc-fg-muted);
		font-size: 10px;
	}
	.ps-new {
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
		font-weight: 700;
		opacity: 0;
		transform: scale(0.9);
		transition:
			opacity 260ms ease,
			transform 260ms ease;
	}
	.ps-new .ps-time {
		color: inherit;
		opacity: 0.8;
	}
	.ps-on {
		opacity: 1;
		transform: none;
	}
</style>
