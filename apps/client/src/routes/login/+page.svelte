<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Alert, AuthScreen } from '@facile/muse';
	import { backend } from '$lib/backend';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	let failed = $state(false);

	onMount(async () => {
		failed = page.url.searchParams.has('error');
		try {
			await backend.me();
			goto('/calendar');
		} catch {}
	});
</script>

<svelte:head>
	<title>{t('login.title')}</title>
</svelte:head>

<AuthScreen
	appName="Agenda"
	href="{backend.apiBaseUrl}/auth/oidc"
	onLocaleChange={rememberLocale}
>
	{#snippet logo()}
		<iconify-icon icon="solar:calendar-add-bold-duotone" width="40" height="40" class="block text-fc-fg"></iconify-icon>
	{/snippet}
	{#snippet tagline()}
		{t('login.tagline')}
	{/snippet}
	{#snippet notice()}
		{#if failed}
			<Alert tone="danger">{t('login.failed')}</Alert>
		{/if}
	{/snippet}
</AuthScreen>
