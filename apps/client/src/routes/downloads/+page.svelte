<script lang="ts">
	import { DownloadSection, LandingFooter, LandingHeader, SignInWithFacile } from '@facile/muse';
	import { backend } from '$lib/backend';
	import { CHECKSUMS_HREF, RELEASES_HREF, cliCommands } from '$lib/downloads';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	const commands = $derived(cliCommands());
</script>

<svelte:head>
	<title>{t('downloads.title')} · Agenda</title>
</svelte:head>

<div class="min-h-dvh overflow-x-clip bg-fc-page text-fc-fg">
	<LandingHeader appName="Agenda" icon="solar:calendar-add-bold-duotone" skipHref="#main">
		{#snippet actions()}
			<SignInWithFacile href="{backend.apiBaseUrl}/auth/oidc" />
		{/snippet}
	</LandingHeader>

	<main id="main" class="mx-auto max-w-5xl px-4 pt-16 pb-20 sm:px-6 sm:pt-20">
		<h1 class="text-center font-fc-title text-fc-3xl font-bold tracking-tight">{t('downloads.title')}</h1>
		<p class="mt-3 text-center text-fc-sm text-fc-fg-muted">{t('landing.download.text')}</p>
		<DownloadSection class="mt-10" {commands} releasesHref={RELEASES_HREF} checksumsHref={CHECKSUMS_HREF} />
	</main>

	<LandingFooter
		appName="Agenda"
		icon="solar:calendar-add-bold-duotone"
		tagline={t('login.tagline')}
		legal={{ onLocaleChange: rememberLocale }}
	/>
</div>
