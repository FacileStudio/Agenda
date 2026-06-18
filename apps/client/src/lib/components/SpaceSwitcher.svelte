<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend, type SpaceItem } from '$lib/backend';
	import {
		getSpaceContext,
		setSpaceContext,
		isPersonal,
		type SpaceContext
	} from '$lib/space-context.svelte';

	let spaces = $state<SpaceItem[]>([]);
	let open = $state(false);

	const current = $derived(getSpaceContext());
	const currentLabel = $derived(
		current === 'personal' ? 'Personnel' : current.name
	);

	onMount(async () => {
		try {
			spaces = await backend.listSpaces();
		} catch {
			spaces = [];
		}
	});

	function select(ctx: SpaceContext) {
		setSpaceContext(ctx);
		open = false;
	}
</script>

<div class="relative px-3 pb-2">
	<button
		onclick={() => (open = !open)}
		class="flex w-full cursor-pointer items-center gap-2.5 rounded-lg border border-border/60 bg-muted/30 px-3 py-2 text-left text-sm transition-colors hover:bg-muted"
	>
		<iconify-icon
			icon={isPersonal() ? 'solar:user-circle-bold-duotone' : 'solar:users-group-rounded-bold-duotone'}
			width="16"
			class="shrink-0 text-muted-foreground"
		></iconify-icon>
		<span class="flex-1 truncate font-medium">{currentLabel}</span>
		<iconify-icon
			icon="solar:alt-arrow-down-linear"
			width="14"
			class="shrink-0 text-muted-foreground transition-transform {open ? 'rotate-180' : ''}"
		></iconify-icon>
	</button>

	{#if open}
		<div class="absolute left-3 right-3 z-50 mt-1 max-h-64 overflow-auto rounded-lg border border-border bg-background p-1 shadow-lg">
			<button
				onclick={() => select('personal')}
				class="flex w-full cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors {isPersonal() ? 'bg-foreground text-background' : 'text-foreground hover:bg-muted'}"
			>
				<iconify-icon icon="solar:user-circle-bold-duotone" width="16" class="shrink-0"></iconify-icon>
				<span class="flex-1">Personnel</span>
			</button>

			{#each spaces as space (space.id)}
				{@const active = !isPersonal() && current !== 'personal' && current.spaceId === space.id}
				<button
					onclick={() => select({ spaceId: space.id, name: space.name, role: space.role })}
					class="flex w-full cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-left text-sm transition-colors {active ? 'bg-foreground text-background' : 'text-foreground hover:bg-muted'}"
				>
					<iconify-icon icon="solar:users-group-rounded-bold-duotone" width="16" class="shrink-0"></iconify-icon>
					<span class="flex-1 truncate">{space.name}</span>
				</button>
			{/each}

			<div class="border-t border-border p-1">
				<button
					onclick={() => { open = false; goto('/spaces'); }}
					class="flex w-full cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
				>
					<iconify-icon icon="solar:settings-minimalistic-linear" width="16" class="shrink-0"></iconify-icon>
					<span>Gérer les espaces</span>
				</button>
			</div>
		</div>
	{/if}
</div>
