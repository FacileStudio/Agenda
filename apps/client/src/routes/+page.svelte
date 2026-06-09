<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { backend } from '$lib/backend';
	import { Button } from '$lib/components/ui/button';

	let ready = $state(false);

	onMount(async () => {
		try {
			await backend.me();
			goto('/mail');
			return;
		} catch {}
		ready = true;
	});
</script>

<svelte:head>
	<title>Courrier — Self-hosted email for creative studios</title>
</svelte:head>

{#if ready}
	<div class="flex min-h-screen flex-col bg-background text-foreground">
		<header class="flex items-center justify-between px-8 py-6">
			<div class="flex items-center gap-3">
				<iconify-icon icon="solar:letter-bold-duotone" width="28" class="text-foreground"></iconify-icon>
				<span class="text-xl font-bold font-heading tracking-tight">Courrier</span>
			</div>
			<div class="flex items-center gap-3">
				<a href="/login">
					<Button variant="ghost" size="sm">Log in</Button>
				</a>
				<a href="/login?tab=register">
					<Button size="sm">Get started</Button>
				</a>
			</div>
		</header>

		<main class="flex flex-1 flex-col items-center justify-center px-6 pb-24">
			<div class="mx-auto max-w-2xl text-center">
				<div class="mb-8 flex justify-center">
					<iconify-icon icon="solar:letter-bold-duotone" width="72" class="text-foreground/80"></iconify-icon>
				</div>

				<h1 class="text-5xl font-bold font-heading tracking-tight leading-[1.1] sm:text-6xl">
					Your mail.<br />Your server.
				</h1>

				<p class="mx-auto mt-6 max-w-lg text-lg text-muted-foreground leading-relaxed">
					A self-hosted email client for creative studios. Connect your IMAP accounts, read and send from a clean interface. No cloud, no tracking, no compromise.
				</p>

				<div class="mt-10 flex items-center justify-center gap-4">
					<a href="/login?tab=register">
						<Button size="lg" class="px-8">Get started</Button>
					</a>
					<a href="/login">
						<Button variant="outline" size="lg" class="px-8">Log in</Button>
					</a>
				</div>
			</div>
		</main>

		<footer class="border-t border-border px-8 py-6">
			<div class="mx-auto flex max-w-5xl items-center justify-between">
				<p class="text-xs text-muted-foreground">
					© {new Date().getFullYear()} Courrier by <a href="https://facile.studio" class="underline underline-offset-2 hover:text-foreground transition-colors" target="_blank" rel="noopener">Facile</a>.
				</p>
				<p class="text-xs text-muted-foreground">
					Part of the Facile Suite.
				</p>
			</div>
		</footer>
	</div>
{/if}
