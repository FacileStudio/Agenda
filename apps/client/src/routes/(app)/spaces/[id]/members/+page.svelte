<script lang="ts">
	import { page } from '$app/stores';
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
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
	<div class="border-b px-6 pt-6 pb-4">
		<div class="flex items-center gap-3">
			<button onclick={() => goto(`/spaces/${spaceId}`)} class="cursor-pointer text-muted-foreground hover:text-foreground" aria-label="Retour à l'espace">
				<iconify-icon icon="solar:alt-arrow-left-linear" width="20"></iconify-icon>
			</button>
			<div>
				<h1 class="text-2xl font-semibold">Membres</h1>
				<p class="mt-1 text-sm text-muted-foreground">{space?.name ?? ''}</p>
			</div>
		</div>
	</div>

	<div class="flex-1 overflow-auto p-6">
		{#if loading}
			<div class="flex items-center justify-center py-12 text-muted-foreground">
				<iconify-icon icon="solar:refresh-linear" width="20" class="animate-spin"></iconify-icon>
			</div>
		{:else}
			<div class="max-w-2xl space-y-6">
				{#if canManage}
					<form onsubmit={addMember} class="flex items-end gap-3">
						<div class="flex-1 space-y-1.5">
							<Label for="add-email">Ajouter un membre</Label>
							<Input id="add-email" type="email" bind:value={addEmail} placeholder="email@exemple.com" required />
						</div>
						<div class="w-32 space-y-1.5">
							<Label for="add-role">Rôle</Label>
							<select
								id="add-role"
								bind:value={addRole}
								class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
							>
								<option value="member">Membre</option>
								<option value="admin">Admin</option>
							</select>
						</div>
						<Button type="submit" disabled={addBusy || !addEmail.trim()} class="cursor-pointer gap-2 shrink-0">
							<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
							{addBusy ? 'Ajout…' : 'Ajouter'}
						</Button>
					</form>
				{/if}

				<div class="flex flex-col gap-2">
					{#each members as member (member.user_id)}
						<div class="flex items-center gap-3 rounded-xl border border-border/70 bg-card p-3">
							{#if member.avatar_url}
								<img
									src={member.avatar_url}
									alt={member.name || member.email}
									class="h-8 w-8 shrink-0 rounded-full border border-border object-cover"
								/>
							{:else}
								<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-border bg-foreground text-xs font-semibold text-background">
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
									class="h-8 rounded-md border border-input bg-transparent px-2 text-xs shadow-xs focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
								>
									<option value="member">Membre</option>
									<option value="admin">Admin</option>
									<option value="owner">Propriétaire</option>
								</select>
							{:else}
								<span class="shrink-0 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
									{roleLabelMap[member.role] ?? member.role}
								</span>
							{/if}

							{#if canManage && member.user_id !== currentUserId && member.role !== 'owner'}
								<Button
									variant="ghost"
									size="sm"
									onclick={() => removeMember(member.user_id)}
									class="cursor-pointer shrink-0 text-muted-foreground hover:text-destructive"
								>
									<iconify-icon icon="solar:trash-bin-2-linear" width="14"></iconify-icon>
								</Button>
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
