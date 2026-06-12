<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Separator } from '$lib/components/ui/separator';
	import { toast } from 'svelte-sonner';
	import { backend, type UserProfile, type CalendarItem, type ApiTokenStatus } from '$lib/backend';
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
	let avatarFailed = $state(false);

	// Reset the fallback when the avatar URL changes (e.g. after a profile sync).
	$effect(() => {
		void app.user?.avatar_url;
		avatarFailed = false;
	});

	let createOpen = $state(false);
	let managedCalendar = $state<CalendarItem | null>(null);
	let manageOpen = $state(false);

	// CalDAV / API token state
	let apiToken = $state<ApiTokenStatus | null>(null);
	let newToken = $state<string | null>(null);
	let tokenCopied = $state(false);
	let tokenBusy = $state(false);
	let tokenName = $state('CalDAV');

	const isSSOUser = $derived(app.user?.avatar_source === 'oidc');
	const serverOrigin = $derived(backend.baseUrl || (browser ? window.location.origin : ''));
	const caldavPath = $derived(`/dav/${app.user?.email ?? ''}`);
	const caldavUrl = $derived(`${serverOrigin}${caldavPath}`);

	onMount(async () => {
		try { apiToken = await backend.getApiToken(); } catch {}
	});

	async function generateToken() {
		tokenBusy = true;
		newToken = null;
		try {
			const res = await backend.createApiToken(tokenName || 'CalDAV');
			newToken = res.token;
			apiToken = { has_token: true, name: res.name, created_at: res.created_at };
		} catch (e: unknown) {
			toast.error(e instanceof Error ? e.message : 'Erreur.');
		} finally {
			tokenBusy = false;
		}
	}

	async function revokeToken() {
		if (!confirm('Révoquer le token ? Les clients CalDAV utilisant ce token seront déconnectés.')) return;
		tokenBusy = true;
		try {
			await backend.deleteApiToken();
			apiToken = { has_token: false };
			newToken = null;
		} catch (e: unknown) {
			toast.error(e instanceof Error ? e.message : 'Erreur.');
		} finally {
			tokenBusy = false;
		}
	}

	async function copyToken(value: string) {
		await navigator.clipboard.writeText(value);
		tokenCopied = true;
		setTimeout(() => (tokenCopied = false), 2000);
	}

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
					{#if app.user?.avatar_url && !avatarFailed}
						<img
							src={app.user.avatar_url}
							alt={displayName}
							class="h-20 w-20 rounded-full border border-border object-cover"
							onerror={() => (avatarFailed = true)}
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
										{cal.role === 'writer' || cal.role === 'admin' ? 'Éditeur' : 'Lecteur'}
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
						Synchronisez vos calendriers avec Apple Calendar, iOS, DAVx⁵ ou tout client CalDAV.
					</p>
				</div>

				<!-- Credentials bloc -->
				<div class="rounded-xl border border-border bg-card divide-y divide-border">
					<!-- Server -->
					<div class="flex items-center gap-3 px-4 py-3">
						<span class="w-28 shrink-0 text-xs font-medium text-muted-foreground">Serveur</span>
						<code class="min-w-0 flex-1 truncate text-xs">{serverOrigin}</code>
						<Button variant="ghost" size="icon" class="cursor-pointer shrink-0 size-7"
							onclick={() => { copyToken(serverOrigin); toast.success('Copié !'); }}>
							<iconify-icon icon="solar:copy-linear" width="14"></iconify-icon>
						</Button>
					</div>
					<!-- Path (Apple Calendar advanced) -->
					<div class="flex items-center gap-3 px-4 py-3">
						<span class="w-28 shrink-0 text-xs font-medium text-muted-foreground">Chemin</span>
						<code class="min-w-0 flex-1 truncate text-xs">{caldavPath}</code>
						<Button variant="ghost" size="icon" class="cursor-pointer shrink-0 size-7"
							onclick={() => { copyToken(caldavPath); toast.success('Copié !'); }}>
							<iconify-icon icon="solar:copy-linear" width="14"></iconify-icon>
						</Button>
					</div>
					<!-- Full URL (DAVx5, Thunderbird) -->
					<div class="flex items-center gap-3 px-4 py-3">
						<span class="w-28 shrink-0 text-xs font-medium text-muted-foreground">URL complète</span>
						<code class="min-w-0 flex-1 truncate text-xs">{caldavUrl}</code>
						<Button variant="ghost" size="icon" class="cursor-pointer shrink-0 size-7"
							onclick={() => { copyToken(caldavUrl); toast.success('Copié !'); }}>
							<iconify-icon icon="solar:copy-linear" width="14"></iconify-icon>
						</Button>
					</div>
					<!-- Username -->
					<div class="flex items-center gap-3 px-4 py-3">
						<span class="w-28 shrink-0 text-xs font-medium text-muted-foreground">Identifiant</span>
						<code class="min-w-0 flex-1 truncate text-xs">{app.user?.email ?? ''}</code>
						<Button variant="ghost" size="icon" class="cursor-pointer shrink-0 size-7"
							onclick={() => { copyToken(app.user?.email ?? ''); toast.success('Copié !'); }}>
							<iconify-icon icon="solar:copy-linear" width="14"></iconify-icon>
						</Button>
					</div>
					<!-- Password -->
					<div class="flex items-center gap-3 px-4 py-3">
						<span class="w-28 shrink-0 text-xs font-medium text-muted-foreground">Mot de passe</span>
						{#if isSSOUser}
							<span class="min-w-0 flex-1 text-xs text-amber-600 dark:text-amber-400">
								→ token API ci-dessous
							</span>
						{:else}
							<span class="min-w-0 flex-1 text-xs text-muted-foreground">Votre mot de passe Agenda</span>
						{/if}
					</div>
				</div>

				{#if isSSOUser}
					<div class="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-amber-700 dark:text-amber-400 flex items-start gap-2">
						<iconify-icon icon="solar:info-circle-linear" width="14" class="mt-0.5 shrink-0"></iconify-icon>
						<span>Vous êtes connecté via SSO — vous n'avez pas de mot de passe. Générez un <strong>token API</strong> ci-dessous et utilisez-le comme mot de passe CalDAV.</span>
					</div>
				{/if}

				<!-- API token section -->
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<iconify-icon icon="solar:key-linear" width="16" class="text-muted-foreground"></iconify-icon>
							<span class="text-sm font-medium">Token API</span>
							{#if isSSOUser}
								<span class="rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-400">Requis</span>
							{/if}
						</div>
					</div>

					{#if newToken}
						<div class="rounded-lg border border-green-500/30 bg-green-500/5 p-4 space-y-2">
							<p class="text-xs font-medium text-green-700 dark:text-green-400 flex items-center gap-1.5">
								<iconify-icon icon="solar:check-circle-linear" width="14"></iconify-icon>
								Token créé — copiez-le maintenant, il ne sera plus affiché.
							</p>
							<div class="flex items-center gap-2">
								<Input value={newToken} readonly class="font-mono text-xs" />
								<Button variant="outline" size="sm" class="cursor-pointer shrink-0 gap-1.5"
									onclick={() => copyToken(newToken!)}>
									<iconify-icon icon={tokenCopied ? 'solar:check-circle-linear' : 'solar:copy-linear'} width="14"></iconify-icon>
									{tokenCopied ? 'Copié' : 'Copier'}
								</Button>
							</div>
						</div>
					{/if}

					{#if apiToken?.has_token}
						<div class="flex items-center justify-between rounded-lg border border-border px-4 py-3">
							<div class="flex items-center gap-3">
								<iconify-icon icon="solar:key-linear" width="16" class="text-muted-foreground"></iconify-icon>
								<div>
									<p class="text-sm font-medium">{apiToken.name ?? 'Token'}</p>
									{#if apiToken.created_at}
										<p class="text-xs text-muted-foreground">Créé le {new Date(apiToken.created_at).toLocaleDateString('fr-FR')}</p>
									{/if}
								</div>
							</div>
							<Button variant="destructive" size="sm" onclick={revokeToken} disabled={tokenBusy}
								class="cursor-pointer gap-1.5">
								<iconify-icon icon="solar:trash-bin-2-linear" width="14"></iconify-icon>
								Révoquer
							</Button>
						</div>
					{:else}
						<div class="flex items-center gap-2">
							<Input bind:value={tokenName} placeholder="Nom du token (ex : MacBook)" class="flex-1" />
							<Button onclick={generateToken} disabled={tokenBusy} class="cursor-pointer gap-2 shrink-0">
								<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
								{tokenBusy ? 'Génération…' : 'Générer'}
							</Button>
						</div>
					{/if}
				</div>

				<!-- Client-specific instructions -->
				<Separator />
				<div class="space-y-3">
					<p class="text-sm font-medium">Instructions par client</p>

					<div class="rounded-xl border border-border divide-y divide-border text-sm">
						<!-- iOS -->
						<details class="group">
							<summary class="flex cursor-pointer items-center justify-between px-4 py-3 text-sm font-medium select-none">
								<span class="flex items-center gap-2">
									<iconify-icon icon="solar:iphone-linear" width="16" class="text-muted-foreground"></iconify-icon>
									iOS — Calendrier
								</span>
								<iconify-icon icon="solar:alt-arrow-down-linear" width="14" class="text-muted-foreground transition-transform group-open:rotate-180"></iconify-icon>
							</summary>
							<ol class="list-decimal px-4 pb-4 pt-2 pl-10 space-y-1 text-muted-foreground">
								<li>Réglages → Calendrier → Comptes → Ajouter un compte → Autre</li>
								<li>Ajouter un compte CalDAV</li>
								<li>Serveur : <code class="text-foreground">{serverOrigin.replace(/^https?:\/\//, '')}</code></li>
								<li>Nom d'utilisateur : votre email</li>
								<li>Mot de passe : {isSSOUser ? 'le token API ci-dessus' : 'votre mot de passe'}</li>
							</ol>
						</details>

						<!-- macOS Apple Calendar -->
						<details class="group">
							<summary class="flex cursor-pointer items-center justify-between px-4 py-3 text-sm font-medium select-none">
								<span class="flex items-center gap-2">
									<iconify-icon icon="solar:monitor-linear" width="16" class="text-muted-foreground"></iconify-icon>
									macOS — Calendrier
								</span>
								<iconify-icon icon="solar:alt-arrow-down-linear" width="14" class="text-muted-foreground transition-transform group-open:rotate-180"></iconify-icon>
							</summary>
							<ol class="list-decimal px-4 pb-4 pt-2 pl-10 space-y-1 text-muted-foreground">
								<li>Calendrier → Réglages → Comptes → <span class="text-foreground">+</span> → CalDAV (Avancé)</li>
								<li>Nom d'utilisateur : votre email</li>
								<li>Mot de passe : {isSSOUser ? 'le token API ci-dessus' : 'votre mot de passe'}</li>
								<li>Adresse du serveur : <code class="text-foreground">{serverOrigin.replace(/^https?:\/\//, '')}</code></li>
								<li>Chemin d'accès : <code class="text-foreground">{caldavPath}</code></li>
								<li>Port : <code class="text-foreground">443</code> — SSL activé</li>
							</ol>
						</details>

						<!-- DAVx5 -->
						<details class="group">
							<summary class="flex cursor-pointer items-center justify-between px-4 py-3 text-sm font-medium select-none">
								<span class="flex items-center gap-2">
									<iconify-icon icon="solar:smartphone-linear" width="16" class="text-muted-foreground"></iconify-icon>
									Android — DAVx⁵
								</span>
								<iconify-icon icon="solar:alt-arrow-down-linear" width="14" class="text-muted-foreground transition-transform group-open:rotate-180"></iconify-icon>
							</summary>
							<ol class="list-decimal px-4 pb-4 pt-2 pl-10 space-y-1 text-muted-foreground">
								<li>DAVx⁵ → <span class="text-foreground">+</span> → Connexion avec URL et nom d'utilisateur</li>
								<li>URL de base : <code class="text-foreground">{caldavUrl}</code></li>
								<li>Nom d'utilisateur : votre email</li>
								<li>Mot de passe : {isSSOUser ? 'le token API ci-dessus' : 'votre mot de passe'}</li>
							</ol>
						</details>

						<!-- Thunderbird -->
						<details class="group">
							<summary class="flex cursor-pointer items-center justify-between px-4 py-3 text-sm font-medium select-none">
								<span class="flex items-center gap-2">
									<iconify-icon icon="solar:letter-linear" width="16" class="text-muted-foreground"></iconify-icon>
									Thunderbird
								</span>
								<iconify-icon icon="solar:alt-arrow-down-linear" width="14" class="text-muted-foreground transition-transform group-open:rotate-180"></iconify-icon>
							</summary>
							<ol class="list-decimal px-4 pb-4 pt-2 pl-10 space-y-1 text-muted-foreground">
								<li>Agenda → Nouveau calendrier → Sur le réseau → CalDAV</li>
								<li>Emplacement : <code class="text-foreground">{caldavUrl}</code></li>
								<li>Identifiant : votre email</li>
								<li>Mot de passe : {isSSOUser ? 'le token API ci-dessus' : 'votre mot de passe'}</li>
							</ol>
						</details>
					</div>
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
