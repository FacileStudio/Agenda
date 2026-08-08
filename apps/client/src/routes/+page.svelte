<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend } from '$lib/backend';
	import { Button, icons } from '@facile/muse';

	let ready = $state(false);
	let ssoOnly = $state(false);

	onMount(async () => {
		try {
			await backend.me();
			goto('/calendar');
			return;
		} catch {}

		try {
			const cfg = await fetch(`${backend.apiBaseUrl}/auth/config`, {
				credentials: 'include'
			}).then((r) => r.json());
			ssoOnly = cfg.sso_only ?? false;
		} catch {}

		ready = true;
	});

	const startHref = $derived(ssoOnly ? '/login' : '/login?tab=register');

	const features = [
		{
			icon: icons.refresh,
			title: 'CalDAV sync',
			description:
				'Apple Calendar, iOS, Thunderbird and DAVx⁵ all talk to it natively. No plugin, no export.'
		},
		{
			icon: icons.usersGroup,
			title: 'Shared calendars',
			description:
				'Invite your studio, set who can read and who can edit, keep personal calendars personal.'
		},
		{
			icon: icons.shield,
			title: 'Self-hosted',
			description:
				'One binary and a Postgres. Your events stay on your server — no cloud, no tracking.'
		}
	];
</script>

<svelte:head>
	<title>Agenda — Self-hosted calendar for creative studios</title>
	<meta
		name="description"
		content="A self-hosted calendar for creative studios. CalDAV sync, shared calendars, no cloud."
	/>
</svelte:head>

{#if ready}
	<div class="min-h-screen bg-background text-foreground">
		<header class="border-b border-border">
			<div class="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
				<div class="flex h-14 items-center gap-3">
					<iconify-icon
						icon="solar:calendar-bold-duotone"
						width="28"
						height="28"
						class="block shrink-0"
					></iconify-icon>
					<span class="font-heading text-2xl font-semibold tracking-tight">Agenda</span>
				</div>
				<div class="flex items-center gap-2">
					<Button variant="ghost" href="/login">Log in</Button>
					<Button href={startHref}>{ssoOnly ? 'Continue with SSO' : 'Get started'}</Button>
				</div>
			</div>
		</header>

		<main>
			<section class="mx-auto max-w-5xl px-6 py-24 text-center">
				<h1 class="font-heading text-5xl font-semibold leading-[1.1] tracking-tight">
					Your calendar.<br />Your server.
				</h1>
				<p class="mx-auto mt-6 max-w-xl text-lg text-muted-foreground">
					Agenda is a self-hosted calendar for creative studios. Plan the week, share it with the
					team, and sync it to every device you already use.
				</p>
				<div class="mt-10 flex justify-center gap-3">
					<Button size="lg" href={startHref} iconRight={icons.arrow}>
						{ssoOnly ? 'Continue with SSO' : 'Start planning'}
					</Button>
					<Button size="lg" variant="outline" href="/login">Log in</Button>
				</div>
			</section>

			<div class="mx-auto max-w-5xl"><hr class="border-border" /></div>

			<section class="mx-auto max-w-5xl px-6 py-20">
				<div class="grid gap-6 md:grid-cols-3">
					{#each features as feature (feature.title)}
						<div class="rounded-fc-md border border-border p-6">
							<div
								class="mb-3 flex size-10 items-center justify-center rounded-fc-md border border-border"
							>
								<iconify-icon icon={feature.icon} width="20" height="20" class="block shrink-0"
								></iconify-icon>
							</div>
							<h3 class="text-base font-semibold">{feature.title}</h3>
							<p class="mt-1.5 text-sm text-muted-foreground">{feature.description}</p>
						</div>
					{/each}
				</div>
			</section>

			<div class="mx-auto max-w-5xl"><hr class="border-border" /></div>

			<section class="mx-auto max-w-5xl px-6 py-20 text-center">
				<h2 class="font-heading text-3xl font-semibold tracking-tight">
					{ssoOnly ? 'Ready to sign in?' : 'Ready to start?'}
				</h2>
				<p class="mt-4 text-muted-foreground">
					{ssoOnly
						? 'Use your organization SSO to access Agenda.'
						: 'Free to use. Self-hosted. No credit card required.'}
				</p>
				<div class="mt-8 flex justify-center">
					<Button size="lg" href={startHref}>
						{ssoOnly ? 'Continue with SSO' : 'Create an account'}
					</Button>
				</div>
			</section>
		</main>

		<footer class="border-t border-border">
			<div class="mx-auto max-w-5xl px-6 py-6 text-center text-sm text-muted-foreground">
				&copy; {new Date().getFullYear()} Agenda by
				<a
					href="https://facile.studio"
					target="_blank"
					rel="noopener"
					class="font-semibold underline underline-offset-2 transition-colors hover:text-foreground"
					>Facile</a
				>.
			</div>
		</footer>
	</div>
{/if}
