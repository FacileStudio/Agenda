<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { backend, type SpaceItem, type SpaceMember } from '$lib/backend';
	import { setSpaceContext } from '$lib/space-context.svelte';

	const spaceId = $derived(Number($page.params.id));

	let space = $state<SpaceItem | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

	const canManage = $derived(space?.role === 'owner' || space?.role === 'admin');
	const isOwner = $derived(space?.role === 'owner');

	const roleLabelMap: Record<string, string> = {
		owner: 'Proprietaire',
		admin: 'Admin',
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
			setSpaceContext({ spaceId: s.id, name: s.name, role: s.role });
		} catch {
			goto('/spaces');
		} finally {
			loading = false;
		}
	});
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
		<div class="border-b px-6 pt-6 pb-4">
			<div class="flex items-center gap-3">
				<button onclick={() => goto('/spaces')} class="cursor-pointer text-muted-foreground hover:text-foreground">
					<iconify-icon icon="solar:alt-arrow-left-linear" width="20"></iconify-icon>
				</button>
				<div class="min-w-0 flex-1">
					<h1 class="truncate text-2xl font-semibold">{space.name}</h1>
					{#if space.description}
						<p class="mt-1 truncate text-sm text-muted-foreground">{space.description}</p>
					{/if}
				</div>
				<span class="shrink-0 rounded-full bg-muted px-2.5 py-0.5 text-xs text-muted-foreground">
					{roleLabelMap[space.role] ?? space.role}
				</span>
			</div>
		</div>

		<div class="flex-1 overflow-auto p-6">
			<div class="max-w-2xl space-y-6">
				<div class="flex items-center justify-between">
					<div>
						<h2 class="text-base font-medium">Membres</h2>
						<p class="text-sm text-muted-foreground">{members.length} membre{members.length !== 1 ? 's' : ''}</p>
					</div>
					<div class="flex gap-2">
						{#if canManage}
							<Button onclick={() => goto(`/spaces/${spaceId}/members`)} variant="outline" class="cursor-pointer gap-2">
								<iconify-icon icon="solar:users-group-rounded-linear" width="16"></iconify-icon>
								Gerer
							</Button>
						{/if}
						{#if isOwner}
							<Button onclick={() => goto(`/spaces/${spaceId}/settings`)} variant="outline" class="cursor-pointer gap-2">
								<iconify-icon icon="solar:settings-minimalistic-linear" width="16"></iconify-icon>
								Parametres
							</Button>
						{/if}
					</div>
				</div>

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
							<span class="shrink-0 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
								{roleLabelMap[member.role] ?? member.role}
							</span>
						</div>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>
