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

	const app = getContext<{
		user: UserProfile | null;
		calendars: CalendarItem[];
		refreshCalendars: () => Promise<void>;
	}>('app');

	let createOpen = $state(false);
	let manageOpen = $state(false);
	let managedCalendar = $state<CalendarItem | null>(null);


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

	function typeLabel(calendar: CalendarItem) {
		if (calendar.space_id) return 'Espace';
		return calendar.is_personal ? 'Personnel' : 'Partagé';
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
		description="Un calendrier personnel se partage par adresse email. Celui d’un espace est partagé avec tous ses membres."
		bare
	>
		{#snippet actions()}
			<Button icon={icons.plus} onclick={() => (createOpen = true)}>Nouveau calendrier</Button>
		{/snippet}

		{#if app.calendars.length === 0}
			<div class="rounded-fc-md bg-fc-component p-6 text-fc-sm text-fc-fg-muted">
				Aucun calendrier pour l’instant. Créez-en un pour commencer à partager.
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
							<td class="text-fc-fg-muted">{typeLabel(calendar)}</td>
							<td>
								<Badge tone={roleTone(calendar.role)}>
									{roleLabels[calendar.role] ?? calendar.role}
								</Badge>
							</td>
							<td class="text-right">
								{#if calendar.space_id}
									<Button
										variant="ghost"
										size="sm"
										icon={icons.space}
										href="/spaces/{calendar.space_id}"
									>
										Membres
									</Button>
								{:else if calendar.role === 'owner'}
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
		description="Chaque espace a son calendrier, partagé avec tous ses membres. Il naît avec l’espace et en porte le nom."
	>
		{#snippet actions()}
			<Button variant="outline" href="/spaces" icon={icons.space}>Tous les espaces</Button>
		{/snippet}

		<SettingsRow
			label="Membres d’un espace"
			description="Inviter, changer un rôle ou retirer quelqu’un se fait dans Registre, pour toutes les apps Facile à la fois. Le calendrier de l’espace suit."
		>
			<Button variant="outline" href="/spaces" icon={icons.usersGroup}>Voir les espaces</Button>
		</SettingsRow>
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
