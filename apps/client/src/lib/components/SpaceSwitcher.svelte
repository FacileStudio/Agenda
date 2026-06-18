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
	let loading = $state(true);

	const current = $derived(getSpaceContext());
	const currentLabel = $derived(
		current === 'personal' ? 'Personnel' : current.name
	);

	onMount(async () => {
		loading = true;
		try {
			spaces = await backend.listSpaces();
		} catch {
			spaces = [];
		}
		loading = false;
	});

	function select(ctx: SpaceContext) {
		setSpaceContext(ctx);
		open = false;
	}

	function handleClickOutside(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.space-switcher')) {
			open = false;
		}
	}

	$effect(() => {
		if (open) {
			document.addEventListener('click', handleClickOutside);
			return () => document.removeEventListener('click', handleClickOutside);
		}
	});
</script>

{#if !loading && spaces.length > 0}
	<div class="space-switcher relative px-3 pb-2">
		<button
			class="flex w-full items-center gap-2.5 rounded-lg border border-border/60 bg-muted/30 px-3 py-2 text-left text-sm transition-colors hover:bg-muted/60"
			onclick={() => open = !open}
		>
			<iconify-icon
				icon={isPersonal() ? 'solar:user-circle-bold-duotone' : 'solar:users-group-rounded-bold-duotone'}
				width="18"
				class="shrink-0 text-muted-foreground"
			></iconify-icon>
			<span class="min-w-0 flex-1 truncate font-medium">{currentLabel}</span>
			<iconify-icon
				icon="solar:alt-arrow-down-linear"
				width="14"
				class="shrink-0 text-muted-foreground transition-transform {open ? 'rotate-180' : ''}"
			></iconify-icon>
		</button>

		{#if open}
			<div class="absolute left-3 right-3 z-50 mt-1 overflow-hidden rounded-lg border border-border bg-background shadow-lg">
				<div class="max-h-64 overflow-auto p-1">
					<button
						class="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm transition-colors {isPersonal() ? 'bg-foreground text-background' : 'text-foreground hover:bg-muted'}"
						onclick={() => select('personal')}
					>
						<iconify-icon icon="solar:user-circle-bold-duotone" width="16" class="shrink-0"></iconify-icon>
						Personnel
					</button>

					{#each spaces as space (space.id)}
						{@const active = !isPersonal() && current !== 'personal' && current.spaceId === space.id}
						<button
							class="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm transition-colors {active ? 'bg-foreground text-background' : 'text-foreground hover:bg-muted'}"
							onclick={() => select({ spaceId: space.id, name: space.name, role: space.role })}
						>
							<iconify-icon icon="solar:users-group-rounded-bold-duotone" width="16" class="shrink-0"></iconify-icon>
							<span class="min-w-0 flex-1 truncate">{space.name}</span>
						</button>
					{/each}
				</div>

				<div class="border-t border-border p-1">
					<a
						href="/spaces"
						class="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
						onclick={() => open = false}
					>
						<iconify-icon icon="solar:settings-linear" width="16" class="shrink-0"></iconify-icon>
						Gérer les espaces
					</a>
				</div>
			</div>
		{/if}
	</div>
{/if}
