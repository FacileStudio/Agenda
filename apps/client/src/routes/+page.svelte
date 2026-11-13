<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend } from '$lib/backend';
	import {
		Button,
		DownloadSection,
		LandingFooter,
		LandingHeader,
		SignInWithFacile,
		icons,
		type CliCommand
	} from '@facile/muse';
	import LandingOffers from '$lib/components/LandingOffers.svelte';
	import WeekDemo from '$lib/components/WeekDemo.svelte';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	const CLI_REPO = 'https://github.com/FacileStudio/agenda-cli';

	let ready = $state(false);

	const signInHref = $derived(`${backend.apiBaseUrl}/auth/oidc`);
	const heroLines = $derived(t('landing.heroTitle').split('|'));

	const commands: CliCommand[] = $derived([
		{ label: t('landing.download.facile'), command: 'facile install agenda' },
		{
			label: t('landing.download.script'),
			command: 'curl -fsSL https://raw.githubusercontent.com/FacileStudio/agenda-cli/main/install.sh | bash'
		}
	]);

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
		{ icon: icons.shield, id: 'hosted' },
		{ icon: 'solar:command-linear', id: 'cli' }
	] as const;
</script>

{#if ready}
	<div class="min-h-dvh overflow-x-clip bg-fc-page text-fc-fg">
		<LandingHeader appName="Agenda" icon="solar:calendar-add-bold-duotone" skipHref="#main">
			{#snippet actions()}
				<SignInWithFacile href={signInHref} />
			{/snippet}
		</LandingHeader>

		<main id="main">
			<section class="mx-auto max-w-5xl px-4 pt-20 pb-12 text-center sm:px-6 sm:pt-24">
				<h1 class="font-fc-title text-fc-3xl font-bold leading-[1.1] tracking-tight">
					{heroLines[0]}<br />{heroLines[1]}
				</h1>
				<p class="mx-auto mt-6 max-w-xl text-fc-md text-fc-fg-muted">{t('landing.heroText')}</p>
				<div class="mt-10 flex flex-wrap items-center justify-center gap-3">
					<SignInWithFacile href={signInHref} size="lg" />
					<Button href="#download" variant="secondary" size="lg" icon={icons.download}>
						{t('landing.getCli')}
					</Button>
				</div>
			</section>

			<section class="mx-auto max-w-5xl px-4 pb-16 sm:px-6">
				<WeekDemo />
			</section>

			<section class="mx-auto max-w-5xl px-4 py-16 sm:px-6">
				<div class="grid gap-4 sm:grid-cols-2 sm:gap-6">
					{#each features as feature (feature.id)}
						<div class="rounded-fc-lg bg-fc-surface p-6">
							<div class="mb-3 flex size-10 items-center justify-center rounded-fc-md bg-fc-component">
								<iconify-icon icon={feature.icon} width="20" height="20" class="block shrink-0"
								></iconify-icon>
							</div>
							<h2 class="text-fc-lg font-bold">{t(`landing.feature.${feature.id}.title`)}</h2>
							<p class="mt-1.5 text-fc-sm text-fc-fg-muted">
								{t(`landing.feature.${feature.id}.text`)}
							</p>
						</div>
					{/each}
				</div>
			</section>

			<section id="download" class="mx-auto max-w-5xl scroll-mt-20 px-4 py-16 sm:px-6">
				<h2 class="text-center font-fc-title text-fc-2xl font-bold tracking-tight">
					{t('landing.download.title')}
				</h2>
				<p class="mt-3 text-center text-fc-sm text-fc-fg-muted">{t('landing.download.text')}</p>
				<DownloadSection
					class="mt-8"
					{commands}
					headingLevel={3}
					releasesHref="{CLI_REPO}/releases"
					checksumsHref="{CLI_REPO}/releases/latest/download/checksums.txt"
				/>
			</section>

			<LandingOffers />

			<section class="mx-auto max-w-5xl px-4 py-20 text-center sm:px-6">
				<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight">{t('landing.ctaTitle')}</h2>
				<p class="mt-4 text-fc-sm text-fc-fg-muted">{t('landing.ctaText')}</p>
				<div class="mt-8 flex justify-center">
					<SignInWithFacile href={signInHref} size="lg" />
				</div>
			</section>
		</main>

		<LandingFooter
			appName="Agenda"
			icon="solar:calendar-add-bold-duotone"
			tagline={t('login.tagline')}
			legal={{ onLocaleChange: rememberLocale }}
		/>
	</div>
{/if}
