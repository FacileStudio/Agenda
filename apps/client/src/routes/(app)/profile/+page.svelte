<script lang="ts">
	import { getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Button } from '$lib/components/ui/button';
	import { toast } from 'svelte-sonner';
	import { backend, type UserProfile } from '$lib/backend';

	const app = getContext<{ user: UserProfile | null; setUser: (u: UserProfile) => void }>('app');

	let syncing = $state(false);

	function getInitials(value: string) {
		const parts = value.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase();
	}

	const displayName = $derived(app.user?.name?.trim() || app.user?.email || '');

	async function syncProfile() {
		syncing = true;
		try {
			const r = await backend.syncProfile();
			if (r.synced) {
				const fresh = await backend.me();
				app.setUser(fresh.user);
				toast.success('Profil synchronise depuis SSO.');
			} else {
				toast.info('Profil deja a jour.');
			}
		} catch {
			toast.error('Echec de la synchronisation.');
		} finally {
			syncing = false;
		}
	}
</script>

<svelte:head>
	<title>Profil — Agenda</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b px-6 pt-6 pb-4">
		<h1 class="text-2xl font-semibold">Profil</h1>
		<p class="mt-1 text-sm text-muted-foreground">Vos informations de compte.</p>
	</div>

	<div class="flex-1 overflow-auto p-6">
		<div class="max-w-md space-y-6">
			<!-- Avatar -->
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
				<div class="space-y-1.5">
					<p class="text-sm font-medium">{displayName}</p>
					{#if app.user?.avatar_source === 'oidc'}
						<p class="flex items-center gap-1.5 text-xs text-muted-foreground">
							<iconify-icon icon="solar:check-circle-linear" width="14" class="text-green-500"></iconify-icon>
							Votre photo vient du SSO — changez-la dans Porte, elle se met a jour ici en quelques minutes.
						</p>
					{/if}
					{#if app.user?.avatar_source}
						<Button
							variant="outline"
							size="sm"
							onclick={syncProfile}
							disabled={syncing}
							class="cursor-pointer gap-2"
						>
							<iconify-icon icon="solar:refresh-linear" width="14"></iconify-icon>
							{syncing ? 'Synchronisation…' : 'Sync. SSO'}
						</Button>
					{/if}
				</div>
			</div>

			<!-- Fields -->
			<div class="space-y-4">
				<div class="space-y-1.5">
					<Label for="profile-name">Nom</Label>
					<Input id="profile-name" value={app.user?.name ?? ''} disabled />
				</div>
				<div class="space-y-1.5">
					<Label for="profile-email">Email</Label>
					<Input id="profile-email" value={app.user?.email ?? ''} disabled />
				</div>
			</div>

			<Button
				variant="outline"
				class="cursor-pointer gap-2"
				onclick={() => goto('/settings')}
			>
				<iconify-icon icon="solar:settings-linear" width="16"></iconify-icon>
				Voir les parametres
			</Button>
		</div>
	</div>
</div>
