<script lang="ts">
	import { page } from '$app/stores';
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { toast } from 'svelte-sonner';
	import { backend, type SpaceItem, type SpaceMember, type UserProfile } from '$lib/backend';

	const app = getContext<{ user: UserProfile | null }>('app');
	const spaceId = $derived(Number($page.params.id));

	let space = $state<SpaceItem | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);
	let addEmail = $state('');
	let addRole = $state('member');
	let addBusy = $state(false);
	let leaveBusy = $state(false);

	const canManage = $derived(space?.role === 'owner' || space?.role === 'admin');
	const isOwner = $derived(space?.role === 'owner');

	function roleBadgeClass(role: string): string {
		switch (role) {
			case 'owner':
				return 'bg-amber-500/10 text-amber-600';
			case 'admin':
				return 'bg-blue-500/10 text-blue-600';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}

	const roleLabelMap: Record<string, string> = {
		owner: 'Propriétaire',
		admin: 'Administrateur',
		member: 'Membre'
	};

	async function loadData() {
		try {
			const [s, m] = await Promise.all([
				backend.getSpace(spaceId),
				backend.listSpaceMembers(spaceId)
			]);
			space = s;
			members = m;
		} catch {
			goto('/spaces');
		} finally {
			loading = false;
		}
	}

	onMount(loadData);

	async function addMember(e: Event) {
		e.preventDefault();
		if (!addEmail.trim()) return;
		addBusy = true;
		try {
			await backend.addSpaceMember(spaceId, addEmail.trim(), addRole);
			addEmail = '';
			addRole = 'member';
			toast.success('Membre ajouté.');
			await loadData();
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : 'Erreur.');
		} finally {
			addBusy = false;
		}
	}

	async function removeMember(userId: number) {
		if (!confirm('Retirer ce membre de l\'espace ?')) return;
		try {
			await backend.removeSpaceMember(spaceId, userId);
			toast.success('Membre retiré.');
			await loadData();
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : 'Erreur.');
		}
	}

	async function changeRole(userId: number, newRole: string) {
		try {
			await backend.updateSpaceMemberRole(spaceId, userId, newRole);
			toast.success('Rôle mis à jour.');
			await loadData();
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : 'Erreur.');
		}
	}

	async function leaveSpace() {
		if (!confirm('Quitter cet espace ?')) return;
		leaveBusy = true;
		try {
			await backend.leaveSpace(spaceId);
			toast.success('Vous avez quitté l\'espace.');
			goto('/spaces');
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : 'Erreur.');
		} finally {
			leaveBusy = false;
		}
	}

	const currentUserId = $derived(app.user ? Number(app.user.id) : 0);
</script>

<svelte:head>
	<title>Membres — {space?.name ?? 'Espace'} — Agenda</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<div class="flex items-center gap-3">
			<button onclick={() => goto(`/spaces/${spaceId}`)} class="cursor-pointer text-muted-foreground hover:text-foreground" aria-label="Retour à l'espace">
				<iconify-icon icon="solar:arrow-left-linear" width="20"></iconify-icon>
			</button>
			<div>
				<h1 class="text-lg font-semibold">Membres</h1>
				<p class="mt-1 text-sm text-muted-foreground">{space?.name ?? ''}</p>
			</div>
		</div>
	</div>

	<div class="flex-1 overflow-auto p-4 md:p-8">
		{#if loading}
			<div class="flex items-center justify-center py-12 text-muted-foreground">
				<iconify-icon icon="solar:refresh-linear" width="20" class="animate-spin"></iconify-icon>
			</div>
		{:else}
			<div class="max-w-2xl space-y-6">
				{#if canManage}
					<form onsubmit={addMember} class="flex items-end gap-3">
						<div class="flex-1 space-y-1.5">
							<label for="add-email" class="text-sm font-medium">Ajouter un membre</label>
							<input
								id="add-email"
								type="email"
								bind:value={addEmail}
								placeholder="email@exemple.com"
								required
								class="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							/>
						</div>
						<div class="w-32 space-y-1.5">
							<label for="add-role" class="text-sm font-medium">Rôle</label>
							<select
								id="add-role"
								bind:value={addRole}
								class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
							>
								<option value="member">Membre</option>
								<option value="admin">Admin</option>
							</select>
						</div>
						<Button type="submit" disabled={addBusy || !addEmail.trim()} class="cursor-pointer shrink-0 gap-2">
							<iconify-icon icon="solar:add-circle-linear" width="16"></iconify-icon>
							{addBusy ? 'Ajout…' : 'Ajouter'}
						</Button>
					</form>
				{/if}

				<div class="grid gap-2">
					{#each members as member (member.user_id)}
						<div class="flex items-center gap-3 rounded-lg border border-border p-3">
							{#if member.avatar_url}
								<img
									src={member.avatar_url}
									alt={member.name || member.email}
									class="h-9 w-9 shrink-0 rounded-full border border-border object-cover"
								/>
							{:else}
								<div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-border bg-foreground text-xs font-semibold text-background">
									{(member.name || member.email).slice(0, 2).toUpperCase()}
								</div>
							{/if}
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm font-medium">{member.name || member.email}</p>
								{#if member.name}
									<p class="truncate text-xs text-muted-foreground">{member.email}</p>
								{/if}
							</div>

							{#if isOwner && member.user_id !== currentUserId}
								<select
									value={member.role}
									onchange={(e) => changeRole(member.user_id, (e.target as HTMLSelectElement).value)}
									class="h-10 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
								>
									<option value="member">Membre</option>
									<option value="admin">Admin</option>
									<option value="owner">Propriétaire</option>
								</select>
							{:else}
								<span class="shrink-0 rounded-full px-2.5 py-0.5 text-xs {roleBadgeClass(member.role)}">
									{roleLabelMap[member.role] ?? member.role}
								</span>
							{/if}

							{#if canManage && member.user_id !== currentUserId && member.role !== 'owner'}
								<button
									onclick={() => removeMember(member.user_id)}
									class="shrink-0 cursor-pointer rounded-md p-1.5 text-destructive transition-colors hover:bg-destructive/10"
								>
									<iconify-icon icon="solar:trash-bin-2-linear" width="16"></iconify-icon>
								</button>
							{/if}
						</div>
					{/each}
				</div>

				{#if !isOwner}
					<div class="pt-4">
						<Button
							variant="outline"
							onclick={leaveSpace}
							disabled={leaveBusy}
							class="cursor-pointer gap-2 text-destructive hover:bg-destructive/10 hover:text-destructive"
						>
							<iconify-icon icon="solar:logout-2-linear" width="16"></iconify-icon>
							{leaveBusy ? 'Départ…' : 'Quitter l\'espace'}
						</Button>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>
