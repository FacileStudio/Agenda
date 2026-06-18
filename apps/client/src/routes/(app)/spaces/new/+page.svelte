<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
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
			toast.success('Espace cree.');
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
	<div class="border-b px-6 pt-6 pb-4">
		<div class="flex items-center gap-3">
			<button onclick={() => goto('/spaces')} class="cursor-pointer text-muted-foreground hover:text-foreground">
				<iconify-icon icon="solar:alt-arrow-left-linear" width="20"></iconify-icon>
			</button>
			<div>
				<h1 class="text-2xl font-semibold">Nouvel espace</h1>
				<p class="mt-1 text-sm text-muted-foreground">Creez un espace pour collaborer avec votre equipe.</p>
			</div>
		</div>
	</div>

	<div class="flex-1 overflow-auto p-6">
		<form onsubmit={handleSubmit} class="max-w-md space-y-6">
			<div class="space-y-1.5">
				<Label for="space-name">Nom</Label>
				<Input id="space-name" bind:value={name} placeholder="Mon equipe" required />
			</div>
			<div class="space-y-1.5">
				<Label for="space-desc">Description</Label>
				<Input id="space-desc" bind:value={description} placeholder="Description optionnelle" />
			</div>
			<div class="flex gap-3">
				<Button type="submit" disabled={submitting || !name.trim()} class="cursor-pointer gap-2">
					<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
					{submitting ? 'Creation…' : 'Creer'}
				</Button>
				<Button type="button" variant="outline" onclick={() => goto('/spaces')} class="cursor-pointer">
					Annuler
				</Button>
			</div>
		</form>
	</div>
</div>
