<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend } from '$lib/backend';
	import { LandingCta, LandingFooter, LandingHeader, SignInWithFacile, WorksWith, type WorksWithApp } from '@facile/muse';
	import LandingCards from '$lib/components/LandingCards.svelte';
	import LandingDevelopers from '$lib/components/LandingDevelopers.svelte';
	import LandingOffers from '$lib/components/LandingOffers.svelte';
	import WeekDemo from '$lib/components/WeekDemo.svelte';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';
	import { LANDING_ICONS } from '$lib/landing-icons';

	let ready = $state(false);

	const signInHref = $derived(`${backend.apiBaseUrl}/auth/oidc`);

	const apps: WorksWithApp[] = $derived([
		{ name: 'Opus', icon: LANDING_ICONS.opus, line: t('landing.with.opus'), href: 'https://opus.facile.studio' },
		{ name: 'Bulle', icon: LANDING_ICONS.bulle, line: t('landing.with.bulle'), href: 'https://bulle.facile.studio' }
	]);

	$effect(() => {
		document.querySelector('meta[name="description"]')?.setAttribute('content', t('landing.description'));
	});

	onMount(async () => {
		try {
			await backend.me();
			goto('/calendar');
			return;
		} catch {}
		ready = true;
	});
</script>

<svelte:head>
	<title>{t('landing.title')}</title>
</svelte:head>

{#if ready}
	<div class="min-h-dvh overflow-x-clip bg-fc-page text-fc-fg">
		<LandingHeader appName="Agenda" icon={LANDING_ICONS.agenda} skipHref="#main">
			{#snippet actions()}
				<SignInWithFacile href={signInHref} />
			{/snippet}
		</LandingHeader>

		<main id="main">
			<section class="mx-auto max-w-5xl px-4 pt-20 pb-12 text-center sm:px-6 sm:pt-24">
				<h1 class="mx-auto max-w-3xl font-fc-title text-fc-3xl font-bold leading-[1.1] tracking-tight text-balance">
					{t('landing.heroTitle')}
				</h1>
				<p class="mx-auto mt-6 max-w-xl text-fc-md text-fc-fg-muted">{t('landing.heroText')}</p>
				<LandingCta class="mt-10" {signInHref} align="center" />
			</section>

			<section class="mx-auto max-w-5xl px-4 pb-8 sm:px-6">
				<WeekDemo />
			</section>

			<LandingCards />

			<WorksWith class="mx-auto w-full max-w-5xl px-4 py-16 sm:px-6" {apps} description={t('landing.with.line')} />

			<LandingOffers />

			<LandingDevelopers />

			<section class="mx-auto max-w-5xl px-4 py-20 text-center sm:px-6">
				<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight">{t('landing.ctaTitle')}</h2>
				<p class="mt-4 text-fc-sm text-fc-fg-muted">{t('landing.ctaText')}</p>
				<LandingCta class="mt-8" {signInHref} align="center" />
			</section>
		</main>

		<LandingFooter
			appName="Agenda"
			icon={LANDING_ICONS.agenda}
			tagline={t('login.tagline')}
			legal={{ onLocaleChange: rememberLocale }}
		/>
	</div>
{/if}
