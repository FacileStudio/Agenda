<script lang="ts">
	/**
	 * CHARTE §14: the colour scheme lives here and nowhere else — no toggle floating over
	 * every other page. A one-of-N preference is `OptionCards`, not a `Select`.
	 */
	import { OptionCards, SettingsRow, SettingsSection, StatusDot } from '@facile/muse';
	import { theme, type ThemePreference } from '$lib/theme.svelte';
	import { t } from '$lib/i18n/index.svelte';

	const modes = $derived([
		{ value: 'system', label: t('appearance.system'), icon: 'solar:monitor-linear' },
		{ value: 'light', label: t('appearance.light'), icon: 'solar:sun-linear' },
		{ value: 'dark', label: t('appearance.dark'), icon: 'solar:moon-linear' }
	]);

	let mode = $state<string>(theme.preference);
</script>

<svelte:head>
	<title>{t('settings.tab.appearance')} — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<SettingsSection
		title={t('appearance.title')}
		description={t('appearance.titleText')}
	>
		<SettingsRow
			label={t('appearance.scheme')}
			description={t('appearance.schemeText')}
			stacked
		>
			<OptionCards
				options={modes}
				bind:value={mode}
				name="theme-mode"
				label={t('appearance.scheme')}
				onSelect={(next) => theme.set(next as ThemePreference)}
			/>
		</SettingsRow>

		<SettingsRow
			label={t('appearance.current')}
			description={t('appearance.currentText')}
		>
			<StatusDot tone="neutral" label={theme.resolved === 'dark' ? t('appearance.dark') : t('appearance.light')} />
		</SettingsRow>
	</SettingsSection>
</div>
