<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { backend } from '$lib/backend';
	import { cn } from '$lib/utils';

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
			await backend.createCalendar({ name: name.trim(), color, description: description || undefined, echo_url: echoUrl.trim() || undefined });
			onCreated();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Une erreur est survenue.';
		} finally {
			saving = false;
		}
	}

	function handleOpenChange(val: boolean) {
		if (!val) onClose();
	}
</script>

<DialogPrimitive.Root bind:open onOpenChange={handleOpenChange}>
	<DialogPrimitive.Portal>
		<DialogPrimitive.Overlay
			class="data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 fixed inset-0 z-50 bg-black/40 supports-backdrop-filter:backdrop-blur-xs"
		/>
		<DialogPrimitive.Content
			class={cn(
				'data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95',
				'fixed top-1/2 left-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2',
				'rounded-2xl border bg-background p-6 shadow-xl'
			)}
		>
			<div class="mb-5 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Nouveau calendrier</h2>
				<DialogPrimitive.Close
					class="cursor-pointer rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
				>
					<iconify-icon icon="solar:close-circle-linear" width="18"></iconify-icon>
				</DialogPrimitive.Close>
			</div>

			<div class="flex flex-col gap-4">
				<div class="flex flex-col gap-1.5">
					<Label for="cal-name">Nom</Label>
					<Input id="cal-name" bind:value={name} placeholder="Mon calendrier" autofocus />
				</div>

				<div class="flex flex-col gap-1.5">
					<Label>Couleur</Label>
					<div class="flex items-center gap-2 flex-wrap">
						{#each COLORS as c}
							<button
								type="button"
								onclick={() => (color = c)}
								class={cn(
									'size-7 cursor-pointer rounded-full border-2 transition-transform hover:scale-105',
									color === c ? 'border-foreground scale-110' : 'border-transparent'
								)}
								style="background-color: {c}"
							></button>
						{/each}
					</div>
				</div>

				<div class="flex flex-col gap-1.5">
					<Label for="cal-desc">Description</Label>
					<Input id="cal-desc" bind:value={description} placeholder="Description (optionnelle)" />
				</div>

				<div class="flex flex-col gap-1.5">
					<Label for="cal-echo-url">
						<span class="flex items-center gap-1.5">
							<iconify-icon icon="solar:videocamera-record-bold-duotone" width="14" class="text-muted-foreground"></iconify-icon>
							Echo (visioconference)
						</span>
					</Label>
					<Input id="cal-echo-url" bind:value={echoUrl} placeholder="https://echo.facile.studio" />
					<p class="text-xs text-muted-foreground">URL de l'instance Echo pour ce calendrier</p>
				</div>

				{#if error}
					<p class="text-sm text-destructive">{error}</p>
				{/if}
			</div>

			<div class="mt-6 flex justify-end gap-2">
				<Button variant="outline" onclick={onClose} disabled={saving}>Annuler</Button>
				<Button onclick={handleCreate} disabled={saving} class="gap-2">
					<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
					{saving ? 'Creation…' : 'Creer'}
				</Button>
			</div>
		</DialogPrimitive.Content>
	</DialogPrimitive.Portal>
</DialogPrimitive.Root>
