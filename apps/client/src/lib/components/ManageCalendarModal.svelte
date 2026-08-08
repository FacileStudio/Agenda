<script lang="ts">
	import {
		Alert,
		Avatar,
		Badge,
		Button,
		ColorPicker,
		ConfirmModal,
		Divider,
		Field,
		Input,
		Modal,
		Select,
		icons
	} from '@facile/muse';
	import { backend, ApiError, resolveFileUrl, type CalendarItem, type CalendarMember } from '$lib/backend';

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
	let memberToRemove = $state<CalendarMember | null>(null);
	let confirmRemoveOpen = $state(false);
	let confirmDeleteOpen = $state(false);

	const titleId = $props.id();

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
			memberToRemove = null;
			confirmRemoveOpen = false;
			confirmDeleteOpen = false;
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
		if (!calendar) return;
		deleting = true;
		try {
			await backend.deleteCalendar(calendar.id);
			onDeleted();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Erreur.';
			deleting = false;
			throw e;
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

	async function handleRemoveMember() {
		const member = memberToRemove;
		if (!calendar || !member) return;
		removingId = member.user_id;
		try {
			await backend.removeMember(calendar.id, member.user_id);
			await loadMembers();
		} finally {
			removingId = null;
		}
	}

	function roleLabel(role: string) {
		switch (role) {
			case 'owner': return 'Proprietaire';
			case 'admin': return 'Admin';
			case 'writer': return 'Editeur';
			default: return 'Lecteur';
		}
	}

	function roleTone(role: string): 'owner' | 'admin' | 'neutral' {
		if (role === 'owner') return 'owner';
		if (role === 'admin') return 'admin';
		return 'neutral';
	}
</script>

<Modal bind:open size="lg" showClose {onClose} aria-labelledby={titleId}>
	{#snippet header()}
		<div class="flex items-center gap-2">
			{#if editColor}
				<span class="size-3 shrink-0 rounded-full" style="background-color: {editColor}"></span>
			{/if}
			<h2 id={titleId} class="text-fc-lg font-semibold">Gerer le calendrier</h2>
		</div>
	{/snippet}

	{#if calendar?.role === 'owner'}
		<div class="flex flex-col gap-4">
			<Field label="Nom">
				<Input bind:value={editName} placeholder="Nom du calendrier" />
			</Field>

			<div class="flex flex-col gap-1.5">
				<span class="text-fc-sm text-fc-fg">Couleur</span>
				<ColorPicker bind:value={editColor} colors={COLORS} label="Couleur du calendrier" />
			</div>

			<Field label="Description">
				<Input bind:value={editDescription} placeholder="Description (optionnelle)" />
			</Field>

			<Field label="Echo (visioconférence)" helper="URL de l'instance Echo pour ce calendrier">
				<Input bind:value={editEchoUrl} placeholder="https://echo.facile.studio" />
			</Field>

			{#if error}
				<Alert tone="danger">{error}</Alert>
			{/if}

			<div class="flex justify-end">
				<Button icon={icons.check} onclick={handleSave} disabled={saving}>
					{saving ? 'Enregistrement…' : 'Enregistrer'}
				</Button>
			</div>
		</div>

		<Divider class="my-5" />
	{/if}

	<div class="flex flex-col gap-3">
		<div class="flex items-center gap-2">
			<iconify-icon icon={icons.usersGroup} width="16" height="16" class="block size-4 text-fc-fg-muted"
			></iconify-icon>
			<h3 class="text-fc-sm font-medium">Membres</h3>
		</div>

		{#if loadingMembers}
			<p class="text-fc-sm text-fc-fg-muted">Chargement…</p>
		{:else if members.length === 0}
			<p class="text-fc-sm text-fc-fg-muted">Aucun membre partage.</p>
		{:else}
			<div class="flex flex-col divide-y divide-fc-border">
				{#each members as member (member.user_id)}
					<div class="flex items-center gap-3 py-2">
						<Avatar
							size="sm"
							src={member.avatar_url ? resolveFileUrl(member.avatar_url) : undefined}
							name={member.name || member.email}
						/>
						<div class="min-w-0 flex-1">
							<p class="truncate text-fc-sm font-medium text-fc-fg">{member.name || member.email}</p>
							<p class="truncate text-fc-xs text-fc-fg-muted">{member.email}</p>
						</div>
						<Badge tone={roleTone(member.role)}>{roleLabel(member.role)}</Badge>
						{#if calendar?.role === 'owner' && member.role !== 'owner'}
							<Button
								variant="ghost-danger"
								size="sm"
								icon={icons.remove}
								aria-label="Retirer {member.name || member.email}"
								disabled={removingId === member.user_id}
								onclick={() => { memberToRemove = member; confirmRemoveOpen = true; }}
							/>
						{/if}
					</div>
				{/each}
			</div>
		{/if}

		{#if calendar?.role === 'owner'}
			<div class="flex flex-col gap-2 rounded-fc-md bg-fc-surface p-3">
				<p class="text-fc-xs font-medium text-fc-fg-muted">Inviter un membre</p>
				<div class="flex flex-col gap-2 sm:flex-row">
					<Input
						bind:value={inviteEmail}
						placeholder="email@exemple.com"
						type="email"
						aria-label="Email du membre à inviter"
						class="flex-1"
						onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter') handleInvite(); }}
					/>
					<Select bind:value={inviteRole} aria-label="Rôle" class="sm:w-32">
						<option value="reader">Lecteur</option>
						<option value="writer">Editeur</option>
					</Select>
					<Button size="lg" icon={icons.plus} onclick={handleInvite} disabled={inviting}>
						Inviter
					</Button>
				</div>
				{#if inviteError}
					<p class="text-fc-xs text-fc-danger">{inviteError}</p>
				{/if}
			</div>
		{/if}
	</div>

	{#if calendar?.role === 'owner' && !calendar?.is_personal}
		<Divider class="my-5" />
		<div class="flex items-center justify-between gap-3">
			<p class="text-fc-xs text-fc-fg-muted">Zone dangereuse</p>
			<Button
				variant="danger"
				size="sm"
				icon={icons.remove}
				onclick={() => (confirmDeleteOpen = true)}
				disabled={deleting}
			>
				{deleting ? 'Suppression…' : 'Supprimer ce calendrier'}
			</Button>
		</div>
	{/if}
</Modal>

<ConfirmModal
	bind:open={confirmDeleteOpen}
	tone="danger"
	title="Supprimer ce calendrier ?"
	description={`« ${calendar?.name ?? ''} » et tous ses événements seront supprimés pour chaque membre. Cette action est irréversible.`}
	confirmLabel="Supprimer"
	onConfirm={handleDelete}
/>

<ConfirmModal
	bind:open={confirmRemoveOpen}
	tone="danger"
	title="Retirer ce membre ?"
	description={`${memberToRemove?.name || memberToRemove?.email || 'Cette personne'} perdra immédiatement l'accès à ce calendrier.`}
	confirmLabel="Retirer"
	onConfirm={handleRemoveMember}
	onCancel={() => (memberToRemove = null)}
/>
