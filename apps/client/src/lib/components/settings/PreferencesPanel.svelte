<script lang="ts">
	import { PreferencesSection } from '@facile/muse';
	import { theme, type ThemePreference } from '$lib/theme.svelte';
	import { REGISTRE_SPACES_URL } from '$lib/backend';
	import { rememberLocale } from '$lib/i18n/index.svelte';
</script>

<PreferencesSection
	subscription={{ href: REGISTRE_SPACES_URL }}
	theme={theme.mode}
	onThemeChange={(next) => theme.set(next as ThemePreference)}
	onLocaleChange={rememberLocale}
/>
