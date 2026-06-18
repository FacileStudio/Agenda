<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { toast } from 'svelte-sonner';
	import { backend, type SpaceItem } from '$lib/backend';
	import { setSpaceContext } from '$lib/space-context.svelte';

	const spaceId = $derived(Number($page.params.id));

	let space = $state<SpaceItem | null>(null);
	let loading = $state(true);
	let name = $state('');
	let description = $state('');
	let saving = $state(false);
	let deleting = $state(false);

	onMount(async () => {
		try {
			space = await backend.getSpace(spaceId);
			name = space.name;
			description = space.description;
		} catch {
			goto('/spaces');
		} finally {
			loading = false;
		}
	});

	async function handleSave(e: Event) {
		e.preventDefault();
		if (!name.trim()) return;
		saving = true;
		try {
			const updated = await backend.updateSpace(spaceId, { name: name.trim(), description: description.trim() });
			space = updated;
			setSpaceContext({ spaceId: updated.id, name: updated.name, role: updated.role });
			toast.success('Espace mis à jour.');
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : 'Erreur.');
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!confirm('Supprimer cet espace ? Cette action est irréversible.')) return;
		deleting = true;
		try {
			await backend.deleteSpace(spaceId);
			setSpaceContext('personal');
			toast.success('Espace supprimé.');
			goto('/spaces');
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : 'Erreur.');
		} finally {
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>Paramètres — {space?.name ?? 'Espace'} — Agenda</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<div class="flex items-center gap-3">
			<button onclick={() => goto(`/spaces/${spaceId}`)} class="cursor-pointer text-muted-foreground hover:text-foreground" aria-label="Retour à l'espace">
				<iconify-icon icon="solar:arrow-left-linear" width="20"></iconify-icon>
			</button>
			<div>
				<h1 class="text-lg font-semibold">Paramètres</h1>
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
			<div class="max-w-xl space-y-8">
				<form onsubmit={handleSave} class="space-y-6">
					<div class="space-y-1.5">
						<label for="edit-name" class="text-sm font-medium">Nom</label>
						<input
							id="edit-name"
							bind:value={name}
							required
							class="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
					</div>
					<div class="space-y-1.5">
						<label for="edit-desc" class="text-sm font-medium">Description</label>
						<input
							id="edit-desc"
							bind:value={description}
							placeholder="Description optionnelle"
							class="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
						/>
					</div>
					<Button type="submit" disabled={saving || !name.trim()} class="cursor-pointer gap-2">
						<iconify-icon icon="solar:pen-2-linear" width="16"></iconify-icon>
						{saving ? 'Enregistrement…' : 'Enregistrer'}
					</Button>
				</form>

				<div class="border-t border-border pt-8">
					<h2 class="text-lg font-semibold text-destructive">Zone dangereuse</h2>
					<p class="mt-2 text-sm text-muted-foreground">
						La suppression de l'espace est définitive et retire tous les membres.
					</p>
					<Button
						variant="outline"
						onclick={handleDelete}
						disabled={deleting}
						class="mt-4 cursor-pointer gap-2 border-destructive/30 bg-destructive/10 text-destructive hover:bg-destructive/20"
					>
						<iconify-icon icon="solar:trash-bin-2-linear" width="16"></iconify-icon>
						{deleting ? 'Suppression…' : 'Supprimer l\'espace'}
					</Button>
				</div>
			</div>
		{/if}
	</div>
</div>
