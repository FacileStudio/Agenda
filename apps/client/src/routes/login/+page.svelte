<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { backend } from '$lib/backend';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { LogIn, UserPlus, ExternalLink } from 'lucide-svelte';

	let tab = $state<'login' | 'register'>('login');
	let email = $state('');
	let password = $state('');
	let message = $state('');
	let busy = $state(false);
	let ssoOnly = $state(false);
	let oidcEnabled = $state(false);
	let configLoaded = $state(false);

	onMount(async () => {
		try {
			await backend.me();
			goto('/mail');
			return;
		} catch {}

		const raw = $page.url.searchParams.get('tab');
		if (raw === 'register') tab = 'register';

		try {
			const cfg = await fetch(`${backend.baseUrl}/auth/config`, { credentials: 'include' }).then(r => r.json());
			ssoOnly = cfg.sso_only ?? false;
			oidcEnabled = cfg.oidc_enabled ?? false;
			if (ssoOnly) tab = 'login';
		} catch {}
		configLoaded = true;
	});

	async function submit() {
		busy = true;
		message = '';
		try {
			if (tab === 'register') {
				await backend.register(email, password);
			} else {
				await backend.login(email, password);
			}
			goto('/mail');
		} catch (err) {
			message = err instanceof Error ? err.message : 'Something went wrong';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>{!ssoOnly && tab === 'register' ? 'Create account' : 'Log in'} — Courrier</title>
</svelte:head>

<div class="flex min-h-screen">
	<div class="hidden lg:flex lg:w-1/2 flex-col bg-black px-12 py-10">
		<a href="/" class="flex items-center gap-3 mb-auto">
			<iconify-icon icon="solar:letter-bold-duotone" width="28" class="text-white"></iconify-icon>
			<span class="text-xl font-bold font-heading tracking-tight text-white">Courrier</span>
		</a>

		<div class="mb-auto">
			<h2 class="text-4xl font-bold font-heading text-white leading-tight tracking-tight">
				Your mail.<br />Your server.
			</h2>
			<p class="mt-4 text-sm text-white/50 max-w-xs leading-relaxed">
				A clean, self-hosted email client for creative studios.
			</p>
		</div>

		<p class="text-xs text-white/30">
			© {new Date().getFullYear()} Courrier by Facile.
		</p>
	</div>

	<div class="flex w-full lg:w-1/2 flex-col items-center justify-center px-8 py-12 bg-background">
		<div class="w-full max-w-sm">
			<div class="mb-8">
				<h1 class="text-2xl font-bold font-heading tracking-tight text-foreground">
					{!ssoOnly && tab === 'register' ? 'Create account' : 'Welcome back'}
				</h1>
				<p class="mt-1.5 text-sm text-muted-foreground">
					{!ssoOnly && tab === 'register'
						? 'Create an account to get started.'
						: ssoOnly
							? 'Sign in with your organization account to access Courrier.'
							: 'Log in to your Courrier account.'}
				</p>
			</div>

			{#if !configLoaded}
				<div class="h-40"></div>
			{:else}
				{#if !ssoOnly}
					<div class="mb-6 flex rounded-lg border border-border bg-muted p-1 gap-1">
						<button
							class="flex-1 rounded-md py-1.5 text-sm font-medium transition-colors {tab === 'login'
								? 'bg-background text-foreground shadow-sm'
								: 'text-muted-foreground hover:text-foreground'}"
							onclick={() => { tab = 'login'; message = ''; }}
						>
							Log in
						</button>
						<button
							class="flex-1 rounded-md py-1.5 text-sm font-medium transition-colors {tab === 'register'
								? 'bg-background text-foreground shadow-sm'
								: 'text-muted-foreground hover:text-foreground'}"
							onclick={() => { tab = 'register'; message = ''; }}
						>
							Register
						</button>
					</div>

					<form
						onsubmit={(e) => { e.preventDefault(); submit(); }}
						class="space-y-4"
					>
						<div class="space-y-1.5">
							<Label for="email">Email</Label>
							<Input id="email" type="email" bind:value={email} placeholder="you@example.com" required />
						</div>

						<div class="space-y-1.5">
							<Label for="password">Password</Label>
							<Input id="password" type="password" bind:value={password} placeholder="••••••••" required />
						</div>

						{#if message}
							<p class="text-sm text-destructive">{message}</p>
						{/if}

						<Button type="submit" class="w-full gap-2" disabled={busy}>
							{#if tab === 'register'}
								<UserPlus class="h-4 w-4" />
								Create account
							{:else}
								<LogIn class="h-4 w-4" />
								Log in
							{/if}
						</Button>
					</form>
				{/if}

				{#if oidcEnabled}
					{#if !ssoOnly}
						<div class="my-5 flex items-center gap-3">
							<div class="h-px flex-1 bg-border"></div>
							<span class="text-xs text-muted-foreground">or</span>
							<div class="h-px flex-1 bg-border"></div>
						</div>
					{/if}

					<a href="{backend.baseUrl}/auth/oidc" class="block">
						<Button variant="outline" class="w-full gap-2" type="button">
							<ExternalLink class="h-4 w-4" />
							Continue with SSO
						</Button>
					</a>
				{/if}
			{/if}
		</div>
	</div>
</div>
