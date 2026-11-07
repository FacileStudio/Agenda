<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { theme } from '$lib/theme.svelte';
	import { Toaster, provideLocale } from '@facile/muse';
	import { bindLocale, localeSource } from '$lib/i18n/index.svelte';

	if (browser) {
		theme.restore();
		void import('iconify-icon');
	}

	let { children } = $props();

	const i18n = provideLocale(localeSource);
	bindLocale(() => i18n.locale);
</script>

{@render children()}
<Toaster />
