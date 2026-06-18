<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { toast } from 'svelte-sonner';
	import { backend } from '$lib/backend';
	import { setSpaceContext } from '$lib/space-context.svelte';

	let name = $state('');
	let description = $state('');
	let submitting = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!name.trim()) return;
		submitting = true;
		try {
			const space = await backend.createSpace({ name: name.trim(), description: description.trim() });
			setSpaceContext({ spaceId: space.id, name: space.name, role: space.role });
			toast.success('Espace créé.');
			goto(`/spaces/${space.id}`);
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : 'Erreur.');
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Nouvel espace — Agenda</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b border-border px-4 py-4 md:px-8 md:py-5">
		<div class="flex items-center gap-3">
			<button onclick={() => goto('/spaces')} class="cursor-pointer text-muted-foreground hover:text-foreground" aria-label="Retour aux espaces">
				<iconify-icon icon="solar:arrow-left-linear" width="20"></iconify-icon>
			</button>
			<div>
				<h1 class="text-lg font-semibold">Nouvel espace</h1>
				<p class="mt-1 text-sm text-muted-foreground">Créez un espace pour collaborer avec votre équipe.</p>
			</div>
		</div>
	</div>

	<div class="flex-1 overflow-auto p-4 md:p-8">
		<form onsubmit={handleSubmit} class="max-w-xl space-y-6">
			<div class="space-y-1.5">
				<label for="space-name" class="text-sm font-medium">Nom</label>
				<input
					id="space-name"
					bind:value={name}
					placeholder="Mon équipe"
					required
					class="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
				/>
			</div>
			<div class="space-y-1.5">
				<label for="space-desc" class="text-sm font-medium">Description</label>
				<input
					id="space-desc"
					bind:value={description}
					placeholder="Description optionnelle"
					class="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
				/>
			</div>
			<div class="flex gap-3">
				<Button type="submit" disabled={submitting || !name.trim()} class="cursor-pointer gap-2">
					<iconify-icon icon="solar:add-circle-linear" width="16"></iconify-icon>
					{submitting ? 'Création…' : 'Créer'}
				</Button>
				<Button type="button" variant="outline" onclick={() => goto('/spaces')} class="cursor-pointer">
					Annuler
				</Button>
			</div>
		</form>
	</div>
</div>
