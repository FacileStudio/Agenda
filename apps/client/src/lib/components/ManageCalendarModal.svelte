<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Select from '$lib/components/ui/select';
	import { Separator } from '$lib/components/ui/separator';
	import { backend, ApiError, type CalendarItem, type CalendarMember } from '$lib/backend';
	import { cn } from '$lib/utils';

	let {
		open = $bindable(false),
		calendar,
		onUpdated,
		onDeleted,
		onClose
	}: {
		open: boolean;
		calendar: CalendarItem | null;
		onUpdated: () => void;
		onDeleted: () => void;
		onClose: () => void;
	} = $props();

	const COLORS = [
		'#ef4444', '#f97316', '#eab308', '#22c55e',
		'#06b6d4', '#3b82f6', '#8b5cf6', '#ec4899'
	];

	let members = $state<CalendarMember[]>([]);
	let loadingMembers = $state(false);
	let inviteEmail = $state('');
	let inviteRole = $state('reader');
	let inviting = $state(false);
	let inviteError = $state('');
	let removingId = $state<number | null>(null);

	let editName = $state('');
	let editColor = $state('');
	let editDescription = $state('');
	let editEchoUrl = $state('');
	let saving = $state(false);
	let deleting = $state(false);
	let error = $state('');

	$effect(() => {
		if (open && calendar) {
			editName = calendar.name;
			editColor = calendar.color;
			editDescription = calendar.description || '';
			editEchoUrl = calendar.echo_url || '';
			error = '';
			inviteEmail = '';
			inviteRole = 'reader';
			inviteError = '';
			members = [];
			loadMembers();
		}
	});

	async function loadMembers() {
		if (!calendar) return;
		loadingMembers = true;
		try {
			members = await backend.listMembers(calendar.id);
		} catch {
			members = [];
		} finally {
			loadingMembers = false;
		}
	}

	async function handleSave() {
		if (!calendar || !editName.trim()) { error = 'Le nom est requis.'; return; }
		saving = true; error = '';
		try {
			await backend.updateCalendar(calendar.id, {
				name: editName.trim(),
				color: editColor,
				description: editDescription || undefined,
				echo_url: editEchoUrl.trim() || undefined
			});
			onUpdated();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Erreur.';
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!calendar || !confirm(`Supprimer le calendrier "${calendar.name}" ? Cette action est irreversible.`)) return;
		deleting = true;
		try {
			await backend.deleteCalendar(calendar.id);
			onDeleted();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Erreur.';
			deleting = false;
		}
	}

	async function handleInvite() {
		const email = inviteEmail.trim().toLowerCase();
		if (!calendar || !email) { inviteError = 'Email requis.'; return; }
		if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) { inviteError = 'Adresse email invalide.'; return; }
		inviting = true; inviteError = '';
		try {
			await backend.shareCalendar(calendar.id, email, inviteRole);
			inviteEmail = '';
			await loadMembers();
		} catch (e: unknown) {
			inviteError = inviteErrorMessage(e);
		} finally {
			inviting = false;
		}
	}

	function inviteErrorMessage(e: unknown): string {
		if (e instanceof ApiError) {
			if (e.status === 404) return 'Aucun compte trouve avec cet email. La personne doit d\'abord se connecter a Agenda.';
			if (e.status === 403) return 'Vous n\'avez pas la permission de partager ce calendrier.';
			if (e.message.includes('yourself')) return 'Vous ne pouvez pas vous inviter vous-meme.';
			if (e.status === 400) return 'Invitation invalide. Verifiez l\'email et le role.';
		}
		return 'L\'invitation a echoue. Reessayez.';
	}

	async function handleRemoveMember(userId: number) {
		if (!calendar) return;
		removingId = userId;
		try {
			await backend.removeMember(calendar.id, userId);
			await loadMembers();
		} catch {
		} finally {
			removingId = null;
		}
	}

	function handleOpenChange(val: boolean) {
		if (!val) onClose();
	}

	function roleLabel(role: string) {
		switch (role) {
			case 'owner': return 'Proprietaire';
			case 'admin': return 'Admin';
			case 'writer': return 'Editeur';
			default: return 'Lecteur';
		}
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
				'fixed top-1/2 left-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2',
				'rounded-2xl border bg-background p-6 shadow-xl max-h-[90vh] overflow-y-auto'
			)}
		>
			<!-- Header -->
			<div class="mb-5 flex items-center justify-between">
				<div class="flex items-center gap-2">
					{#if editColor}
						<span class="size-3 rounded-full flex-shrink-0" style="background-color: {editColor}"></span>
					{/if}
					<h2 class="text-lg font-semibold">Gerer le calendrier</h2>
				</div>
				<DialogPrimitive.Close
					class="cursor-pointer rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
				>
					<iconify-icon icon="solar:close-circle-linear" width="18"></iconify-icon>
				</DialogPrimitive.Close>
			</div>

			<!-- Edit form (owners only) -->
			{#if calendar?.role === 'owner'}
				<div class="flex flex-col gap-4">
					<div class="flex flex-col gap-1.5">
						<Label for="edit-name">Nom</Label>
						<Input id="edit-name" bind:value={editName} placeholder="Nom du calendrier" />
					</div>

					<div class="flex flex-col gap-1.5">
						<Label>Couleur</Label>
						<div class="flex items-center gap-2 flex-wrap">
							{#each COLORS as c}
								<button
									type="button"
									onclick={() => (editColor = c)}
									class={cn(
										'size-7 cursor-pointer rounded-full border-2 transition-transform hover:scale-105',
										editColor === c ? 'border-foreground scale-110' : 'border-transparent'
									)}
									style="background-color: {c}"
								></button>
							{/each}
						</div>
					</div>

					<div class="flex flex-col gap-1.5">
						<Label for="edit-desc">Description</Label>
						<Input id="edit-desc" bind:value={editDescription} placeholder="Description (optionnelle)" />
					</div>

					<div class="flex flex-col gap-1.5">
						<Label for="edit-echo-url">
							<span class="flex items-center gap-1.5">
								<iconify-icon icon="solar:videocamera-record-bold-duotone" width="14" class="text-muted-foreground"></iconify-icon>
								Echo (visioconference)
							</span>
						</Label>
						<Input id="edit-echo-url" bind:value={editEchoUrl} placeholder="https://echo.facile.studio" />
						<p class="text-xs text-muted-foreground">URL de l'instance Echo pour ce calendrier</p>
					</div>

					{#if error}
						<p class="text-sm text-destructive">{error}</p>
					{/if}

					<div class="flex justify-end">
						<Button onclick={handleSave} disabled={saving} class="gap-2">
							<iconify-icon icon="solar:check-circle-linear" width="16"></iconify-icon>
							{saving ? 'Enregistrement…' : 'Enregistrer'}
						</Button>
					</div>
				</div>

				<Separator class="my-5" />
			{/if}

			<!-- Members section -->
			<div class="flex flex-col gap-3">
				<div class="flex items-center gap-2">
					<iconify-icon icon="solar:users-group-rounded-linear" width="16" class="text-muted-foreground"></iconify-icon>
					<h3 class="text-sm font-medium">Membres</h3>
				</div>

				{#if loadingMembers}
					<p class="text-sm text-muted-foreground">Chargement…</p>
				{:else if members.length === 0}
					<p class="text-sm text-muted-foreground">Aucun membre partage.</p>
				{:else}
					<div class="flex flex-col gap-1">
						{#each members as member (member.user_id)}
							<div class="flex items-center gap-3 rounded-lg border border-border/60 px-3 py-2">
								<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold">
									{(member.name || member.email).slice(0, 2).toUpperCase()}
								</div>
								<div class="min-w-0 flex-1">
									<p class="truncate text-sm font-medium">{member.name || member.email}</p>
									<p class="truncate text-xs text-muted-foreground">{member.email}</p>
								</div>
								<span class="shrink-0 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
									{roleLabel(member.role)}
								</span>
								{#if calendar?.role === 'owner' && member.role !== 'owner'}
									<button
										onclick={() => handleRemoveMember(member.user_id)}
										disabled={removingId === member.user_id}
										class="flex size-7 cursor-pointer items-center justify-center shrink-0 rounded-full bg-destructive text-white hover:bg-destructive/90 disabled:opacity-50"
										title="Retirer ce membre"
									>
										<iconify-icon icon="solar:trash-bin-2-linear" width="14"></iconify-icon>
									</button>
								{/if}
							</div>
						{/each}
					</div>
				{/if}

				<!-- Invite form (owners only) -->
				{#if calendar?.role === 'owner'}
					<div class="flex flex-col gap-2 rounded-lg border border-dashed border-border p-3">
						<p class="text-xs font-medium text-muted-foreground">Inviter un membre</p>
						<div class="flex gap-2">
							<Input
								bind:value={inviteEmail}
								placeholder="email@exemple.com"
								type="email"
								class="flex-1"
								onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter') handleInvite(); }}
							/>
							<Select.Root type="single" value={inviteRole} onValueChange={(v) => { if (v) inviteRole = v; }}>
								<Select.Trigger class="w-28">
									{roleLabel(inviteRole)}
								</Select.Trigger>
								<Select.Content>
									<Select.Item value="reader">Lecteur</Select.Item>
									<Select.Item value="writer">Editeur</Select.Item>
								</Select.Content>
							</Select.Root>
							<Button onclick={handleInvite} disabled={inviting} class="gap-1.5 shrink-0">
								<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
								Inviter
							</Button>
						</div>
						{#if inviteError}
							<p class="text-xs text-destructive">{inviteError}</p>
						{/if}
					</div>
				{/if}
			</div>

			<!-- Footer: delete calendar -->
			{#if calendar?.role === 'owner' && !calendar?.is_personal}
				<Separator class="my-5" />
				<div class="flex items-center justify-between">
					<p class="text-xs text-muted-foreground">Zone dangereuse</p>
					<Button
						variant="destructive"
						size="sm"
						onclick={handleDelete}
						disabled={deleting}
						class="gap-2"
					>
						<iconify-icon icon="solar:trash-bin-2-linear" width="14"></iconify-icon>
						{deleting ? 'Suppression…' : 'Supprimer ce calendrier'}
					</Button>
				</div>
			{/if}
		</DialogPrimitive.Content>
	</DialogPrimitive.Portal>
</DialogPrimitive.Root>
