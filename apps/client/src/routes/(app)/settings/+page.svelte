<script lang="ts">
	import { getContext } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Separator } from '$lib/components/ui/separator';
	import { toast } from 'svelte-sonner';
	import { backend, type UserProfile, type CalendarItem } from '$lib/backend';
	import CreateCalendarModal from '$lib/components/CreateCalendarModal.svelte';
	import ManageCalendarModal from '$lib/components/ManageCalendarModal.svelte';

	const app = getContext<{
		user: UserProfile | null;
		calendars: CalendarItem[];
		setUser: (u: UserProfile) => void;
		refreshCalendars: () => Promise<void>;
	}>('app');

	type Tab = 'profile' | 'calendars' | 'caldav';

	let activeTab = $state<Tab>('profile');
	let syncing = $state(false);

	let createOpen = $state(false);
	let managedCalendar = $state<CalendarItem | null>(null);
	let manageOpen = $state(false);

	const tabs: { id: Tab; label: string; icon: string }[] = [
		{ id: 'profile', label: 'Profil', icon: 'solar:user-linear' },
		{ id: 'calendars', label: 'Calendriers', icon: 'solar:calendar-linear' },
		{ id: 'caldav', label: 'CalDAV', icon: 'solar:server-2-linear' }
	];

	function getInitials(value: string) {
		const parts = value.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase();
	}

	const displayName = $derived(app.user?.name?.trim() || app.user?.email || '');

	async function syncProfile() {
		syncing = true;
		try {
			const r = await backend.syncProfile();
			if (r.synced) {
				const fresh = await backend.me();
				app.setUser(fresh.user);
				toast.success('Profil synchronisé depuis SSO.');
			} else {
				toast.info('Profil déjà à jour.');
			}
		} catch {
			toast.error('Échec de la synchronisation.');
		} finally {
			syncing = false;
		}
	}

	function openManage(cal: CalendarItem) {
		managedCalendar = cal;
		manageOpen = true;
	}

	const caldavBase = $derived(`${backend.baseUrl}/dav`);
</script>

<svelte:head>
	<title>Paramètres — Agenda</title>
</svelte:head>

<div class="flex h-full flex-col">
	<div class="border-b px-6 pt-6 pb-0">
		<h1 class="text-2xl font-semibold">Paramètres</h1>
		<p class="mt-1 text-sm text-muted-foreground">Gérez votre compte et vos calendriers.</p>

		<!-- Tabs -->
		<div class="mt-4 flex gap-1">
			{#each tabs as tab}
				<button
					onclick={() => (activeTab = tab.id)}
					class="flex cursor-pointer items-center gap-2 border-b-2 px-3 py-2 text-sm font-medium transition-colors {activeTab === tab.id
						? 'border-foreground text-foreground'
						: 'border-transparent text-muted-foreground hover:text-foreground'}"
				>
					<iconify-icon icon={tab.icon} width="15"></iconify-icon>
					{tab.label}
				</button>
			{/each}
		</div>
	</div>

	<div class="flex-1 overflow-auto p-6">
		<!-- Profile tab -->
		{#if activeTab === 'profile'}
			<div class="max-w-md space-y-6">
				<!-- Avatar -->
				<div class="flex items-center gap-4">
					{#if app.user?.avatar_url}
						<img
							src={app.user.avatar_url}
							alt={displayName}
							class="h-20 w-20 rounded-full border border-border object-cover"
						/>
					{:else}
						<div
							class="flex h-20 w-20 items-center justify-center rounded-full border border-border bg-foreground text-xl font-semibold text-background"
						>
							{getInitials(displayName)}
						</div>
					{/if}
					<div class="space-y-1">
						<p class="text-sm font-medium">{displayName}</p>
						{#if app.user?.avatar_source === 'oidc'}
							<p class="flex items-center gap-1.5 text-xs text-muted-foreground">
								<iconify-icon icon="solar:check-circle-linear" width="14" class="text-green-500"></iconify-icon>
								Avatar synchronisé depuis SSO
							</p>
						{/if}
						{#if app.user?.avatar_source}
							<Button
								variant="outline"
								size="sm"
								onclick={syncProfile}
								disabled={syncing}
								class="mt-1 cursor-pointer gap-2"
							>
								<iconify-icon icon="solar:refresh-linear" width="14"></iconify-icon>
								{syncing ? 'Synchronisation…' : 'Sync. profil SSO'}
							</Button>
						{/if}
					</div>
				</div>

				<Separator />

				<!-- Fields -->
				<div class="space-y-4">
					<div class="space-y-1.5">
						<Label for="s-name">Nom</Label>
						<Input id="s-name" value={app.user?.name ?? ''} disabled />
					</div>
					<div class="space-y-1.5">
						<Label for="s-email">Email</Label>
						<Input id="s-email" value={app.user?.email ?? ''} disabled />
					</div>
				</div>

				{#if app.user?.avatar_source === 'oidc'}
					<p class="rounded-lg border border-dashed border-border p-3 text-xs text-muted-foreground">
						Votre profil est géré par votre fournisseur SSO. Les modifications doivent être effectuées là-bas puis synchronisées ici.
					</p>
				{/if}
			</div>

		<!-- Calendars tab -->
		{:else if activeTab === 'calendars'}
			<div class="max-w-2xl space-y-4">
				<div class="flex items-center justify-between">
					<div>
						<h2 class="text-base font-medium">Vos calendriers</h2>
						<p class="text-sm text-muted-foreground">Créez et gérez vos calendriers partagés.</p>
					</div>
					<Button onclick={() => (createOpen = true)} class="cursor-pointer gap-2">
						<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
						Nouveau calendrier
					</Button>
				</div>

				<div class="flex flex-col gap-2">
					{#each app.calendars as cal (cal.id)}
						<div class="flex items-center gap-3 rounded-xl border border-border/70 bg-card p-4">
							<span
								class="size-4 shrink-0 rounded-full border border-black/10"
								style="background-color: {cal.color}"
							></span>
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm font-medium">{cal.name}</p>
								{#if cal.description}
									<p class="truncate text-xs text-muted-foreground">{cal.description}</p>
								{/if}
							</div>
							<div class="flex items-center gap-2 shrink-0">
								{#if cal.is_personal}
									<span class="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">Personnel</span>
								{/if}
								{#if cal.role !== 'owner'}
									<span class="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
										{cal.role === 'editor' ? 'Éditeur' : 'Lecteur'}
									</span>
								{/if}
								{#if cal.role === 'owner'}
									<Button
										variant="ghost"
										size="sm"
										onclick={() => openManage(cal)}
										class="cursor-pointer gap-1.5 text-muted-foreground hover:text-foreground"
									>
										<iconify-icon icon="solar:settings-minimalistic-linear" width="14"></iconify-icon>
										Gérer
									</Button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</div>

		<!-- CalDAV tab -->
		{:else if activeTab === 'caldav'}
			<div class="max-w-lg space-y-6">
				<div>
					<h2 class="text-base font-medium">Connexion CalDAV</h2>
					<p class="mt-1 text-sm text-muted-foreground">
						Connectez vos clients CalDAV (Apple Calendar, Thunderbird, DAVx⁵…) à vos calendriers.
					</p>
				</div>

				<div class="space-y-4">
					<div class="space-y-1.5">
						<Label>URL du serveur</Label>
						<div class="flex items-center gap-2">
							<Input value="{caldavBase}/{app.user?.email ?? ''}" readonly class="font-mono text-xs" />
							<Button
								variant="outline"
								size="sm"
								class="cursor-pointer shrink-0"
								onclick={() => {
									navigator.clipboard.writeText(`${caldavBase}/${app.user?.email ?? ''}`);
									toast.success('Copié !');
								}}
							>
								<iconify-icon icon="solar:copy-linear" width="16"></iconify-icon>
							</Button>
						</div>
					</div>

					<div class="space-y-1.5">
						<Label>Identifiant</Label>
						<Input value={app.user?.email ?? ''} readonly />
					</div>

					<div class="space-y-1.5">
						<Label>Mot de passe</Label>
						<Input value="Votre mot de passe Agenda" disabled class="text-muted-foreground" />
					</div>
				</div>

				<div class="rounded-xl border border-dashed border-border p-4 space-y-2">
					<p class="text-sm font-medium flex items-center gap-2">
						<iconify-icon icon="solar:info-circle-linear" width="16" class="text-muted-foreground"></iconify-icon>
						Instructions
					</p>
					<ol class="list-decimal pl-4 space-y-1 text-sm text-muted-foreground">
						<li>Dans votre client CalDAV, ajoutez un nouveau compte.</li>
						<li>Entrez l'URL du serveur ci-dessus.</li>
						<li>Utilisez votre email comme identifiant.</li>
						<li>Entrez votre mot de passe Agenda.</li>
					</ol>
				</div>
			</div>
		{/if}
	</div>
</div>

<CreateCalendarModal
	bind:open={createOpen}
	onCreated={() => { createOpen = false; app.refreshCalendars(); toast.success('Calendrier créé.'); }}
	onClose={() => (createOpen = false)}
/>

<ManageCalendarModal
	bind:open={manageOpen}
	calendar={managedCalendar}
	onUpdated={() => { app.refreshCalendars(); toast.success('Calendrier mis à jour.'); }}
	onDeleted={() => { manageOpen = false; managedCalendar = null; app.refreshCalendars(); toast.success('Calendrier supprimé.'); }}
	onClose={() => { manageOpen = false; managedCalendar = null; }}
/>
