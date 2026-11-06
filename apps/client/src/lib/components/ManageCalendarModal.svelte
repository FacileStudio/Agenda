<script lang="ts">
	import {
		Alert,
		Avatar,
		Badge,
		Button,
		SwatchPicker,
		ConfirmModal,
		Divider,
		Field,
		Input,
		Modal,
		Select,
		icons
	} from '@facile/muse';
	import { backend, ApiError, resolveFileUrl, type CalendarItem, type CalendarMember } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

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
		if (!calendar || !editName.trim()) { error = t('calendarModal.nameRequired'); return; }
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
			error = e instanceof Error ? e.message : t('common.error');
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
			error = e instanceof Error ? e.message : t('common.error');
			deleting = false;
			throw e;
		}
	}

	async function handleInvite() {
		const email = inviteEmail.trim().toLowerCase();
		if (!calendar || !email) { inviteError = t('member.emailRequired'); return; }
		if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) { inviteError = t('member.emailInvalid'); return; }
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
			if (e.status === 404) return t('member.err404');
			if (e.status === 403) return t('member.err403');
			if (e.message.includes('yourself')) return t('member.errSelf');
			if (e.status === 400) return t('member.err400');
		}
		return t('member.errFailed');
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
			case 'owner': return t('role.owner');
			case 'admin': return t('role.admin');
			case 'writer': return t('role.writer');
			default: return t('role.reader');
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
			<h2 id={titleId} class="text-fc-lg font-semibold">{t('calendarModal.manageTitle')}</h2>
		</div>
	{/snippet}

	{#if calendar?.role === 'owner'}
		<div class="flex flex-col gap-4">
			<Field label={t('common.name')}>
				<Input bind:value={editName} placeholder={t('calendarModal.nameLong')} />
			</Field>

			<div class="flex flex-col gap-1.5">
				<span class="text-fc-sm text-fc-fg">{t('common.color')}</span>
				<SwatchPicker bind:value={editColor} colors={COLORS} label={t('calendarModal.colorLabel')} />
			</div>

			<Field label={t('common.description')}>
				<Input bind:value={editDescription} placeholder={t('calendarModal.descriptionPlaceholder')} />
			</Field>

			<Field label={t('calendarModal.echo')} helper={t('calendarModal.echoHelper')}>
				<Input bind:value={editEchoUrl} placeholder="https://echo.facile.studio" />
			</Field>

			{#if error}
				<Alert tone="danger">{error}</Alert>
			{/if}

			<div class="flex justify-end">
				<Button icon={icons.check} onclick={handleSave} disabled={saving}>
					{saving ? t('common.saving') : t('common.save')}
				</Button>
			</div>
		</div>

		<Divider class="my-5" />
	{/if}

	<div class="flex flex-col gap-3">
		<div class="flex items-center gap-2">
			<iconify-icon icon={icons.usersGroup} width="16" height="16" class="block size-4 text-fc-fg-muted"
			></iconify-icon>
			<h3 class="text-fc-sm font-medium">{t('common.members')}</h3>
		</div>

		{#if loadingMembers}
			<p class="text-fc-sm text-fc-fg-muted">{t('common.loadingDots')}</p>
		{:else if members.length === 0}
			<p class="text-fc-sm text-fc-fg-muted">{t('member.none')}</p>
		{:else}
			<div class="flex flex-col">
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
								aria-label={t('member.removeAria', { name: member.name || member.email })}
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
				<p class="text-fc-xs font-medium text-fc-fg-muted">{t('member.invite')}</p>
				<div class="flex flex-col gap-2 sm:flex-row">
					<Input
						bind:value={inviteEmail}
						placeholder={t('member.emailPlaceholder')}
						type="email"
						aria-label={t('member.emailAria')}
						class="flex-1"
						onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter') handleInvite(); }}
					/>
					<Select bind:value={inviteRole} aria-label={t('member.roleAria')} class="sm:w-32">
						<option value="reader">{t('role.reader')}</option>
						<option value="writer">{t('role.writer')}</option>
					</Select>
					<Button size="lg" icon={icons.plus} onclick={handleInvite} disabled={inviting}>
						{t('member.inviteButton')}
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
			<p class="text-fc-xs text-fc-fg-muted">{t('calendarModal.dangerZone')}</p>
			<Button
				variant="danger"
				size="sm"
				icon={icons.remove}
				onclick={() => (confirmDeleteOpen = true)}
				disabled={deleting}
			>
				{deleting ? t('common.deleting') : t('calendarModal.deleteButton')}
			</Button>
		</div>
	{/if}
</Modal>

<ConfirmModal
	bind:open={confirmDeleteOpen}
	tone="danger"
	title={t('calendarModal.deleteTitle')}
	description={t('calendarModal.deleteText', { name: calendar?.name ?? '' })}
	confirmLabel={t('common.delete')}
	onConfirm={handleDelete}
/>

<ConfirmModal
	bind:open={confirmRemoveOpen}
	tone="danger"
	title={t('member.removeTitle')}
	description={t('member.removeText', {
		name: memberToRemove?.name || memberToRemove?.email || t('member.thisPerson')
	})}
	confirmLabel={t('common.remove')}
	onConfirm={handleRemoveMember}
	onCancel={() => (memberToRemove = null)}
/>
