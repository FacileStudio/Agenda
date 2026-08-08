<script lang="ts">
	/**
	 * CHARTE §14: the colour scheme lives here and nowhere else — no toggle floating over
	 * every other page. A one-of-N preference is `OptionCards`, not a `Select`.
	 */
	import { OptionCards, SettingsRow, SettingsSection, StatusDot } from '@facile/muse';
	import { theme, type ThemePreference } from '$lib/theme.svelte';

	const modes = [
		{ value: 'system', label: 'Système', icon: 'solar:monitor-linear' },
		{ value: 'light', label: 'Clair', icon: 'solar:sun-linear' },
		{ value: 'dark', label: 'Sombre', icon: 'solar:moon-linear' }
	];

	let mode = $state<string>(theme.preference);
</script>

<svelte:head>
	<title>Apparence — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<SettingsSection
		title="Thème"
		description="Stocké dans ce navigateur. Chaque appareil sur lequel vous vous connectez garde son propre choix."
	>
		<SettingsRow
			label="Schéma de couleurs"
			description="Système suit le réglage de votre système d’exploitation et continue de le suivre tant que l’app est ouverte."
			stacked
		>
			<OptionCards
				options={modes}
				bind:value={mode}
				name="theme-mode"
				label="Schéma de couleurs"
				onSelect={(next) => theme.set(next as ThemePreference)}
			/>
		</SettingsRow>

		<SettingsRow
			label="Affiché actuellement"
			description="Ce que la préférence résout à cet instant, une fois le réglage système pris en compte."
		>
			<StatusDot tone="neutral" label={theme.resolved === 'dark' ? 'Sombre' : 'Clair'} />
		</SettingsRow>
	</SettingsSection>
</div>
