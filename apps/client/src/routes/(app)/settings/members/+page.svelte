<script lang="ts">
	/**
	 * Membership in Agenda hangs off two objects: a calendar, which is shared with people
	 * one by one, and a space, which owns its own member list under /spaces. This tab is
	 * the account-level view of the first, and the door to the second.
	 */
	import { getContext } from 'svelte';
	import { Badge, Button, SettingsRow, SettingsSection, Table, icons } from '@facile/muse';
	import { toast } from 'svelte-sonner';
	import type { CalendarItem, UserProfile } from '$lib/backend';
	import { getSpaceContext } from '$lib/space-context.svelte';
	import CreateCalendarModal from '$lib/components/CreateCalendarModal.svelte';
	import ManageCalendarModal from '$lib/components/ManageCalendarModal.svelte';

	const app = getContext<{
		user: UserProfile | null;
		calendars: CalendarItem[];
		refreshCalendars: () => Promise<void>;
	}>('app');

	let createOpen = $state(false);
	let manageOpen = $state(false);
	let managedCalendar = $state<CalendarItem | null>(null);

	const space = $derived(getSpaceContext());
	const spaceName = $derived(space === 'personal' ? 'Personnel' : space.name);

	const roleLabels: Record<string, string> = {
		owner: 'Propriétaire',
		admin: 'Administrateur',
		writer: 'Éditeur',
		reader: 'Lecteur'
	};

	function roleTone(role: string) {
		if (role === 'owner') return 'owner' as const;
		if (role === 'admin') return 'admin' as const;
		return 'neutral' as const;
	}

	function openManage(calendar: CalendarItem) {
		managedCalendar = calendar;
		manageOpen = true;
	}
</script>

<svelte:head>
	<title>Membres — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<SettingsSection
		title="Calendriers"
		description="Un calendrier se partage par adresse email. Le rôle décide de qui peut écrire dedans."
		bare
	>
		{#snippet actions()}
			<Button icon={icons.plus} onclick={() => (createOpen = true)}>Nouveau calendrier</Button>
		{/snippet}

		{#if app.calendars.length === 0}
			<div class="rounded-fc-md bg-fc-component p-6 text-fc-sm text-fc-fg-muted">
				Aucun calendrier dans le contexte <span class="font-medium text-fc-fg">{spaceName}</span>.
				Créez-en un pour commencer à partager.
			</div>
		{:else}
			<Table>
				<thead>
					<tr>
						<th>Calendrier</th>
						<th>Type</th>
						<th>Votre rôle</th>
						<th class="text-right">Action</th>
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
							<td class="text-fc-fg-muted">{calendar.is_personal ? 'Personnel' : 'Partagé'}</td>
							<td>
								<Badge tone={roleTone(calendar.role)}>
									{roleLabels[calendar.role] ?? calendar.role}
								</Badge>
							</td>
							<td class="text-right">
								{#if calendar.role === 'owner'}
									<Button
										variant="ghost"
										size="sm"
										icon={icons.usersGroup}
										onclick={() => openManage(calendar)}
									>
										Gérer
									</Button>
								{:else}
									<span class="text-fc-xs text-fc-fg-muted">Partagé avec vous</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</Table>
		{/if}
	</SettingsSection>

	<SettingsSection
		title="Espaces"
		description="Un espace regroupe des calendriers et leur donne une liste de membres commune."
	>
		{#snippet actions()}
			<Button variant="outline" href="/spaces" icon={icons.folder}>Tous les espaces</Button>
		{/snippet}

		<SettingsRow
			label="Contexte courant"
			description="Les calendriers ci-dessus sont ceux de ce contexte. Changez-le depuis le sélecteur de la barre latérale."
		>
			<Badge tone={space === 'personal' ? 'neutral' : 'accent'}>{spaceName}</Badge>
		</SettingsRow>

		{#if space !== 'personal'}
			<SettingsRow
				label="Membres de l’espace"
				description="Inviter, changer un rôle ou retirer quelqu’un se fait sur l’espace lui-même — les calendriers en héritent."
			>
				<Button variant="outline" href="/spaces/{space.spaceId}/members" icon={icons.usersGroup}>
					Gérer les membres
				</Button>
			</SettingsRow>
		{/if}
	</SettingsSection>
</div>

<CreateCalendarModal
	bind:open={createOpen}
	onCreated={() => {
		createOpen = false;
		app.refreshCalendars();
		toast.success('Calendrier créé.');
	}}
	onClose={() => (createOpen = false)}
/>

<ManageCalendarModal
	bind:open={manageOpen}
	calendar={managedCalendar}
	onUpdated={() => {
		app.refreshCalendars();
		toast.success('Calendrier mis à jour.');
	}}
	onDeleted={() => {
		manageOpen = false;
		managedCalendar = null;
		app.refreshCalendars();
		toast.success('Calendrier supprimé.');
	}}
	onClose={() => {
		manageOpen = false;
		managedCalendar = null;
	}}
/>
