<script lang="ts">
	import { getContext } from 'svelte';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import type { UserProfile } from '$lib/backend';

	const app = getContext<{ user: UserProfile | null }>('app');

	function getInitials(value: string) {
		const parts = value.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase();
	}

	const displayName = $derived(app.user?.name?.trim() || app.user?.email || '');
</script>

<svelte:head>
	<title>Profil — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-0 h-full">
	<div class="px-6 pt-6 pb-0">
		<h1 class="text-2xl font-semibold">Profil</h1>
		<p class="text-sm text-muted-foreground mt-1">Vos informations de compte.</p>
	</div>

	<div class="flex-1 overflow-auto p-6">
		<div class="max-w-md space-y-6">
			<div class="flex items-center gap-4">
				{#if app.user?.avatar_url}
					<img
						src={app.user.avatar_url}
						alt={displayName}
						class="h-20 w-20 rounded-full border border-border object-cover"
					/>
				{:else}
					<div
						class="flex h-20 w-20 items-center justify-center rounded-full border border-border bg-foreground text-xl font-semibold text-background"
					>
						{getInitials(displayName)}
					</div>
				{/if}
				{#if app.user?.avatar_source === 'oidc'}
					<p class="text-xs text-muted-foreground">Avatar synchronisé depuis SSO</p>
				{/if}
			</div>

			<div class="space-y-4">
				<div class="space-y-2">
					<Label for="profile-name">Nom</Label>
					<Input id="profile-name" value={app.user?.name ?? ''} disabled />
				</div>

				<div class="space-y-2">
					<Label for="profile-email">Email</Label>
					<Input id="profile-email" value={app.user?.email ?? ''} disabled />
				</div>
			</div>
		</div>
	</div>
</div>
