<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { Toaster } from '$lib/components/ui/sonner';
	import { theme } from '$lib/theme.svelte';
	import { provideLocale } from '@facile/muse';
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
