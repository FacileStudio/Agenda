<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend } from '$lib/backend';
	import { LandingFooter, LandingHeader, SignInWithFacile, icons } from '@facile/muse';
	import LandingOffers from '$lib/components/LandingOffers.svelte';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	let ready = $state(false);

	const signInHref = $derived(`${backend.apiBaseUrl}/auth/oidc`);
	const heroLines = $derived(t('landing.heroTitle').split('|'));

	onMount(async () => {
		try {
			await backend.me();
			goto('/calendar');
			return;
		} catch {}
		ready = true;
	});

	const features = [
		{ icon: icons.refresh, id: 'caldav' },
		{ icon: icons.usersGroup, id: 'shared' },
		{ icon: icons.shield, id: 'hosted' }
	] as const;
</script>

{#if ready}
	<div class="min-h-screen bg-background text-foreground">
		<LandingHeader appName="Agenda" icon="solar:calendar-add-bold-duotone" skipHref="#main">
			{#snippet actions()}
				<SignInWithFacile href={signInHref} />
			{/snippet}
		</LandingHeader>

		<main id="main">
			<section class="mx-auto max-w-5xl px-6 py-24 text-center">
				<h1 class="font-heading text-5xl font-semibold leading-[1.1] tracking-tight">
					{heroLines[0]}<br />{heroLines[1]}
				</h1>
				<p class="mx-auto mt-6 max-w-xl text-lg text-muted-foreground">{t('landing.heroText')}</p>
				<div class="mt-10 flex justify-center">
					<SignInWithFacile href={signInHref} size="lg" />
				</div>
			</section>

			<section class="mx-auto max-w-5xl px-6 py-20">
				<div class="grid gap-6 md:grid-cols-3">
					{#each features as feature (feature.id)}
						<div class="rounded-fc-md bg-muted/50 p-6">
							<div
								class="mb-3 flex size-10 items-center justify-center rounded-fc-md bg-background"
							>
								<iconify-icon icon={feature.icon} width="20" height="20" class="block shrink-0"
								></iconify-icon>
							</div>
							<h3 class="text-base font-semibold">{t(`landing.feature.${feature.id}.title`)}</h3>
							<p class="mt-1.5 text-sm text-muted-foreground">
								{t(`landing.feature.${feature.id}.text`)}
							</p>
						</div>
					{/each}
				</div>
			</section>

			<section class="mx-auto max-w-5xl px-6 py-20 text-center">
				<h2 class="font-heading text-3xl font-semibold tracking-tight">{t('landing.ctaTitle')}</h2>
				<p class="mt-4 text-muted-foreground">{t('landing.ctaText')}</p>
				<div class="mt-8 flex justify-center">
					<SignInWithFacile href={signInHref} size="lg" />
				</div>
			</section>
			<LandingOffers />
		</main>

		<LandingFooter
			appName="Agenda"
			icon="solar:calendar-add-bold-duotone"
			tagline={t('login.tagline')}
			legal={{ onLocaleChange: rememberLocale }}
		/>
	</div>
{/if}
