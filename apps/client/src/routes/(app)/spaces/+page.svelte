<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { backend, type SpaceItem } from '$lib/backend';
	import { setSpaceContext } from '$lib/space-context.svelte';

	let spaces = $state<SpaceItem[]>([]);
	let loading = $state(true);

	onMount(async () => {
		try {
			spaces = await backend.listSpaces();
		} catch {
			spaces = [];
		} finally {
			loading = false;
		}
	});

	function selectSpace(space: SpaceItem) {
		setSpaceContext({ spaceId: space.id, name: space.name, role: space.role });
		goto(`/spaces/${space.id}`);
	}

	const roleLabelMap: Record<string, string> = {
		owner: 'Proprietaire',
		admin: 'Admin',
		member: 'Membre'
	};
</script>

<svelte:head>
	<title>Espaces — Agenda</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b px-6 pt-6 pb-4">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-semibold">Espaces</h1>
				<p class="mt-1 text-sm text-muted-foreground">Collaborez avec votre equipe dans des espaces partages.</p>
			</div>
			<Button onclick={() => goto('/spaces/new')} class="cursor-pointer gap-2">
				<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
				Nouvel espace
			</Button>
		</div>
	</div>

	<div class="flex-1 overflow-auto p-6">
		{#if loading}
			<div class="flex items-center justify-center py-12 text-muted-foreground">
				<iconify-icon icon="solar:refresh-linear" width="20" class="animate-spin"></iconify-icon>
			</div>
		{:else if spaces.length === 0}
			<div class="flex flex-col items-center justify-center py-16 text-center">
				<iconify-icon icon="solar:users-group-rounded-linear" width="48" class="text-muted-foreground/50"></iconify-icon>
				<p class="mt-4 text-sm text-muted-foreground">Aucun espace pour le moment.</p>
				<Button onclick={() => goto('/spaces/new')} variant="outline" class="mt-4 cursor-pointer gap-2">
					<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
					Creer un espace
				</Button>
			</div>
		{:else}
			<div class="max-w-2xl space-y-2">
				{#each spaces as space (space.id)}
					<button
						onclick={() => selectSpace(space)}
						class="flex w-full cursor-pointer items-center gap-4 rounded-xl border border-border/70 bg-card p-4 text-left transition-colors hover:bg-muted/50"
					>
						<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-foreground/10">
							<iconify-icon icon="solar:users-group-rounded-linear" width="20" class="text-foreground"></iconify-icon>
						</div>
						<div class="min-w-0 flex-1">
							<p class="truncate text-sm font-medium">{space.name}</p>
							{#if space.description}
								<p class="truncate text-xs text-muted-foreground">{space.description}</p>
							{/if}
						</div>
						<span class="shrink-0 rounded-full bg-muted px-2.5 py-0.5 text-xs text-muted-foreground">
							{roleLabelMap[space.role] ?? space.role}
						</span>
						<iconify-icon icon="solar:alt-arrow-right-linear" width="16" class="shrink-0 text-muted-foreground"></iconify-icon>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
