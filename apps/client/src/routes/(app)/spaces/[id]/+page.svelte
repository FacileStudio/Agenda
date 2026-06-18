<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { backend, type SpaceItem, type SpaceMember } from '$lib/backend';
	import { setSpaceContext, getSpaceContext, isPersonal } from '$lib/space-context.svelte';

	const spaceId = $derived(Number($page.params.id));

	let space = $state<SpaceItem | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

	const canManage = $derived(space?.role === 'owner' || space?.role === 'admin');
	const isOwner = $derived(space?.role === 'owner');
	const isCurrentSpace = $derived(!isPersonal() && getSpaceContext() !== 'personal' && (getSpaceContext() as { spaceId: number }).spaceId === spaceId);

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

	onMount(async () => {
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
	});

	function switchToSpace() {
		if (!space) return;
		setSpaceContext({ spaceId: space.id, name: space.name, role: space.role });
	}
</script>

<svelte:head>
	<title>{space?.name ?? 'Espace'} — Agenda</title>
</svelte:head>

<div class="flex h-full flex-col">
	{#if loading}
		<div class="flex flex-1 items-center justify-center">
			<iconify-icon icon="solar:refresh-linear" width="20" class="animate-spin text-muted-foreground"></iconify-icon>
		</div>
	{:else if space}
		<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
			<div class="flex items-center gap-3">
				<button onclick={() => goto('/spaces')} class="cursor-pointer text-muted-foreground hover:text-foreground" aria-label="Retour aux espaces">
					<iconify-icon icon="solar:arrow-left-linear" width="20"></iconify-icon>
				</button>
				<div class="min-w-0 flex-1">
					<h1 class="truncate text-lg font-semibold">{space.name}</h1>
					{#if space.description}
						<p class="mt-1 truncate text-sm text-muted-foreground">{space.description}</p>
					{/if}
				</div>
				<div class="flex shrink-0 items-center gap-2">
					{#if !isCurrentSpace}
						<Button onclick={switchToSpace} class="cursor-pointer gap-2">
							<iconify-icon icon="solar:transfer-horizontal-linear" width="16"></iconify-icon>
							Basculer
						</Button>
					{/if}
					{#if canManage}
						<Button onclick={() => goto(`/spaces/${spaceId}/settings`)} variant="outline" class="cursor-pointer gap-2">
							<iconify-icon icon="solar:settings-minimalistic-linear" width="16"></iconify-icon>
							Paramètres
						</Button>
					{/if}
				</div>
			</div>
		</div>

		<div class="flex-1 overflow-auto p-4 md:p-8">
			<div class="max-w-2xl space-y-6">
				<div class="rounded-lg border border-border/60 bg-muted/20 p-4">
					<p class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Informations</p>
					<div class="mt-3 grid gap-2 text-sm">
						<div class="flex justify-between">
							<span class="text-muted-foreground">Rôle</span>
							<span class="rounded-full px-2.5 py-0.5 text-xs {roleBadgeClass(space.role)}">
								{roleLabelMap[space.role] ?? space.role}
							</span>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">Membres</span>
							<span>{members.length}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">Créé le</span>
							<span>{new Date(space.created_at).toLocaleDateString('fr-FR')}</span>
						</div>
					</div>
				</div>

				<div>
					<div class="flex items-center justify-between">
						<p class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Membres</p>
						{#if canManage}
							<Button onclick={() => goto(`/spaces/${spaceId}/members`)} variant="outline" size="sm" class="cursor-pointer gap-2">
								<iconify-icon icon="solar:users-group-rounded-linear" width="14"></iconify-icon>
								Gérer
							</Button>
						{/if}
					</div>

					<div class="mt-3 grid gap-2">
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
								<span class="shrink-0 rounded-full px-2.5 py-0.5 text-xs {roleBadgeClass(member.role)}">
									{roleLabelMap[member.role] ?? member.role}
								</span>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
