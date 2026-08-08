<script lang="ts">
	import { Alert, Button, ColorPicker, Field, Input, Modal, icons } from '@facile/muse';
	import { backend } from '$lib/backend';
	import { spaceId } from '$lib/space-context.svelte';

	let {
		open = $bindable(false),
		onCreated,
		onClose
	}: {
		open: boolean;
		onCreated: () => void;
		onClose: () => void;
	} = $props();

	const COLORS = [
		'#ef4444', '#f97316', '#eab308', '#22c55e',
		'#06b6d4', '#3b82f6', '#8b5cf6', '#ec4899'
	];

	let name = $state('');
	let color = $state('#3b82f6');
	let description = $state('');
	let echoUrl = $state('');
	let saving = $state(false);
	let error = $state('');

	$effect(() => {
		if (open) {
			name = '';
			color = '#3b82f6';
			description = '';
			echoUrl = '';
			error = '';
		}
	});

	async function handleCreate() {
		if (!name.trim()) { error = 'Le nom est requis.'; return; }
		saving = true; error = '';
		try {
			await backend.createCalendar({ name: name.trim(), color, description: description || undefined, echo_url: echoUrl.trim() || undefined, space_id: spaceId() });
			onCreated();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Une erreur est survenue.';
		} finally {
			saving = false;
		}
	}
</script>

<Modal bind:open title="Nouveau calendrier" showClose {onClose}>
	<div class="flex flex-col gap-4">
		<Field label="Nom">
			<Input bind:value={name} placeholder="Mon calendrier" autofocus />
		</Field>

		<div class="flex flex-col gap-1.5">
			<span class="text-fc-sm text-fc-fg">Couleur</span>
			<ColorPicker bind:value={color} colors={COLORS} label="Couleur du calendrier" />
		</div>

		<Field label="Description">
			<Input bind:value={description} placeholder="Description (optionnelle)" />
		</Field>

		<Field label="Echo (visioconférence)" helper="URL de l'instance Echo pour ce calendrier">
			<Input bind:value={echoUrl} placeholder="https://echo.facile.studio" />
		</Field>

		{#if error}
			<Alert tone="danger">{error}</Alert>
		{/if}
	</div>

	{#snippet footer()}
		<div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
			<Button variant="outline" class="w-full sm:w-auto" onclick={() => (open = false)} disabled={saving}>
				Annuler
			</Button>
			<Button
				class="w-full sm:w-auto"
				icon={icons.plus}
				onclick={handleCreate}
				disabled={saving}
			>
				{saving ? 'Création…' : 'Créer'}
			</Button>
		</div>
	{/snippet}
</Modal>
