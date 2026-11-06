<script lang="ts">
	/**
	 * Membership in Agenda hangs off two objects: a personal calendar, shared with people
	 * one by one, and a space, whose one calendar is shared with every member Registre
	 * lists. This tab manages the first and points at the second.
	 */
	import { getContext } from 'svelte';
	import { Badge, Button, SettingsRow, SettingsSection, Table, icons } from '@facile/muse';
	import { toast } from 'svelte-sonner';
	import type { CalendarItem, UserProfile } from '$lib/backend';
	import CreateCalendarModal from '$lib/components/CreateCalendarModal.svelte';
	import ManageCalendarModal from '$lib/components/ManageCalendarModal.svelte';
	import { t } from '$lib/i18n/index.svelte';

	const app = getContext<{
		user: UserProfile | null;
		calendars: CalendarItem[];
		refreshCalendars: () => Promise<void>;
	}>('app');

	let createOpen = $state(false);
	let manageOpen = $state(false);
	let managedCalendar = $state<CalendarItem | null>(null);

	const roleLabelKeys = {
		owner: 'role.owner',
		admin: 'role.admin',
		writer: 'role.writer',
		reader: 'role.reader'
	} as const;

	function roleLabel(role: string): string {
		return role in roleLabelKeys ? t(roleLabelKeys[role as keyof typeof roleLabelKeys]) : role;
	}

	function roleTone(role: string) {
		if (role === 'owner') return 'owner' as const;
		if (role === 'admin') return 'admin' as const;
		return 'neutral' as const;
	}

	function typeLabel(calendar: CalendarItem) {
		if (calendar.space_id) return t('members.typeSpace');
		return calendar.is_personal ? t('members.typePersonal') : t('members.typeShared');
	}

	function openManage(calendar: CalendarItem) {
		managedCalendar = calendar;
		manageOpen = true;
	}
</script>

<svelte:head>
	<title>{t('settings.tab.members')} — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<SettingsSection
		title={t('members.calendars')}
		description={t('members.calendarsText')}
		bare
	>
		{#snippet actions()}
			<Button icon={icons.plus} onclick={() => (createOpen = true)}>{t('members.newCalendar')}</Button>
		{/snippet}

		{#if app.calendars.length === 0}
			<div class="rounded-fc-md bg-fc-component p-6 text-fc-sm text-fc-fg-muted">
				{t('members.empty')}
			</div>
		{:else}
			<Table>
				<thead>
					<tr>
						<th>{t('members.colCalendar')}</th>
						<th class="hidden sm:table-cell">{t('members.colType')}</th>
						<th>{t('members.colRole')}</th>
						<th class="text-right">{t('api.colAction')}</th>
					</tr>
				</thead>
				<tbody>
					{#each app.calendars as calendar (calendar.id)}
						<tr>
							<td>
								<span class="flex min-w-0 items-center gap-3">
									<span
										class="size-3 shrink-0 rounded-full"
										style="background-color: {calendar.color}"
									></span>
									<span class="flex min-w-0 flex-col">
										<span class="truncate font-medium text-fc-fg">{calendar.name}</span>
										{#if calendar.description}
											<span class="truncate text-fc-xs text-fc-fg-muted">
												{calendar.description}
											</span>
										{/if}
									</span>
								</span>
							</td>
							<td class="hidden text-fc-fg-muted sm:table-cell">{typeLabel(calendar)}</td>
							<td>
								<Badge tone={roleTone(calendar.role)}>
									{roleLabel(calendar.role)}
								</Badge>
							</td>
							<td class="text-right">
								{#if calendar.space_id}
									<Button
										variant="ghost"
										size="sm"
										icon={icons.space}
										href="/spaces/{calendar.space_id}"
										aria-label={t('members.spaceMembers')}
									>
										<span class="hidden sm:inline">{t('common.members')}</span>
									</Button>
								{:else if calendar.role === 'owner'}
									<Button
										variant="ghost"
										size="sm"
										icon={icons.usersGroup}
										onclick={() => openManage(calendar)}
										aria-label={t('members.manageAria', { name: calendar.name })}
									>
										<span class="hidden sm:inline">{t('members.manage')}</span>
									</Button>
								{:else}
									<span class="text-fc-xs text-fc-fg-muted">{t('members.typeShared')}</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</Table>
		{/if}
	</SettingsSection>

	<SettingsSection
		title={t('members.spaces')}
		description={t('members.spacesText')}
	>
		{#snippet actions()}
			<Button variant="outline" href="/spaces" icon={icons.space}>{t('members.allSpaces')}</Button>
		{/snippet}

		<SettingsRow
			label={t('members.spaceRow')}
			description={t('members.spaceRowText')}
		>
			<Button variant="outline" href="/spaces" icon={icons.usersGroup}>{t('members.seeSpaces')}</Button>
		</SettingsRow>
	</SettingsSection>
</div>

<CreateCalendarModal
	bind:open={createOpen}
	onCreated={() => {
		createOpen = false;
		app.refreshCalendars();
		toast.success(t('members.toastCreated'));
	}}
	onClose={() => (createOpen = false)}
/>

<ManageCalendarModal
	bind:open={manageOpen}
	calendar={managedCalendar}
	onUpdated={() => {
		app.refreshCalendars();
		toast.success(t('members.toastUpdated'));
	}}
	onDeleted={() => {
		manageOpen = false;
		managedCalendar = null;
		app.refreshCalendars();
		toast.success(t('members.toastDeleted'));
	}}
	onClose={() => {
		manageOpen = false;
		managedCalendar = null;
	}}
/>
