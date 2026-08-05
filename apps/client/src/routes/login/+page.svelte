<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { backend } from '$lib/backend';

	const inputClass =
		'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50';
	const labelClass = 'text-sm font-medium leading-none';
	const primaryButtonClass =
		'inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50';
	const outlineButtonClass =
		'inline-flex h-10 w-full items-center justify-center rounded-md border border-border bg-background px-4 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50';

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
			goto('/calendar');
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
			goto('/calendar');
		} catch (err) {
			message = err instanceof Error ? err.message : 'Something went wrong';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>{!ssoOnly && tab === 'register' ? 'Create account' : 'Log in'} — Agenda</title>
</svelte:head>

<div class="flex min-h-screen">
	<div class="hidden lg:flex lg:w-1/2 flex-col bg-black px-12 py-10">
		<a href="/" class="flex items-center gap-3 mb-auto">
			<iconify-icon icon="solar:calendar-bold-duotone" width="28" class="text-white"></iconify-icon>
			<span class="text-xl font-bold font-heading tracking-tight text-white">Agenda</span>
		</a>

		<div class="mb-auto">
			<h2 class="text-4xl font-bold font-heading text-white leading-tight tracking-tight">
				Your calendar.<br />Your server.
			</h2>
			<p class="mt-4 text-sm text-white/50 max-w-xs leading-relaxed">
				A clean, self-hosted calendar for creative studios.
			</p>
		</div>

		<p class="text-xs text-white/30">
			© {new Date().getFullYear()} Agenda by Facile.
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
							? 'Sign in with your organization account to access Agenda.'
							: 'Log in to your Agenda account.'}
				</p>
			</div>

			{#if !configLoaded}
				<div class="h-40"></div>
			{:else}
				{#if !ssoOnly}
					<div class="mb-6 flex rounded-lg border border-border bg-muted p-1 gap-1" role="tablist">
						<button
							type="button"
							role="tab"
							aria-selected={tab === 'login'}
							class="flex-1 rounded-md py-1.5 text-sm font-medium transition-colors {tab === 'login'
								? 'bg-background text-foreground shadow-sm'
								: 'text-muted-foreground hover:text-foreground'}"
							onclick={() => { tab = 'login'; message = ''; }}
						>Log in</button>
						<button
							type="button"
							role="tab"
							aria-selected={tab === 'register'}
							class="flex-1 rounded-md py-1.5 text-sm font-medium transition-colors {tab === 'register'
								? 'bg-background text-foreground shadow-sm'
								: 'text-muted-foreground hover:text-foreground'}"
							onclick={() => { tab = 'register'; message = ''; }}
						>Register</button>
					</div>

					<form
						onsubmit={(e) => { e.preventDefault(); submit(); }}
						class="space-y-4"
					>
						<div class="space-y-1.5">
							<label for="email" class={labelClass}>Email</label>
							<input
								id="email"
								type="email"
								bind:value={email}
								placeholder="you@example.com"
								autocomplete="email"
								required
								disabled={busy}
								class={inputClass}
							/>
						</div>

						<div class="space-y-1.5">
							<label for="password" class={labelClass}>Password</label>
							<input
								id="password"
								type="password"
								bind:value={password}
								placeholder="••••••••"
								autocomplete={tab === 'register' ? 'new-password' : 'current-password'}
								required
								disabled={busy}
								class={inputClass}
							/>
						</div>

						{#if message}
							<p class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
								{message}
							</p>
						{/if}

						<button type="submit" disabled={busy} class={primaryButtonClass}>
							{busy
								? tab === 'register' ? 'Creating account…' : 'Logging in…'
								: tab === 'register' ? 'Create account' : 'Log in'}
						</button>
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
						<button type="button" class={outlineButtonClass}>Continue with SSO</button>
					</a>
				{/if}
			{/if}
		</div>
	</div>
</div>
