<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		Badge,
		Button,
		Input,
		ProfileCard,
		SettingsRow,
		SettingsSection,
		icons
	} from '@facile/muse';
	import { toast } from 'svelte-sonner';
	import { backend, type UserProfile } from '$lib/backend';

	const app = getContext<{ user: UserProfile | null; setUser: (u: UserProfile) => void }>('app');

	let syncing = $state(false);
	let loggingOut = $state(false);
	let ssoOnly = $state(false);
	let oidcEnabled = $state(false);

	const displayName = $derived(app.user?.name?.trim() || app.user?.email || '');
	const managedBySSO = $derived(app.user?.avatar_source === 'oidc');
	const signedInWith = $derived(
		ssoOnly ? 'Authentification unique' : oidcEnabled ? 'Mot de passe ou SSO' : 'Mot de passe'
	);
	const createdAt = $derived(
		app.user?.created_at ? new Date(app.user.created_at).toLocaleDateString('fr-FR') : '—'
	);

	onMount(async () => {
		try {
			const response = await fetch(`${backend.apiBaseUrl}/auth/config`, {
				credentials: 'include'
			});
			const config = (await response.json()) as { sso_only: boolean; oidc_enabled: boolean };
			ssoOnly = config.sso_only;
			oidcEnabled = config.oidc_enabled;
		} catch {
			ssoOnly = false;
			oidcEnabled = false;
		}
	});

	async function syncProfile() {
		syncing = true;
		try {
			const result = await backend.syncProfile();
			if (result.synced) {
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

	async function logout() {
		loggingOut = true;
		try {
			await backend.logout();
		} catch {
			toast.warning('Le serveur n’a pas confirmé la déconnexion, la session est fermée ici.');
		}
		await goto('/login');
	}
</script>

<svelte:head>
	<title>Paramètres — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<ProfileCard
		name={displayName}
		email={app.user?.email ?? ''}
		avatar={app.user?.avatar_url ?? ''}
		meta={[
			{ label: 'Connecté via', value: signedInWith },
			{ label: 'Compte créé le', value: createdAt }
		]}
	>
		{#snippet actions()}
			{#if app.user?.avatar_source}
				<Button variant="outline" icon={icons.refresh} disabled={syncing} onclick={syncProfile}>
					{syncing ? 'Synchronisation…' : 'Sync. profil SSO'}
				</Button>
			{/if}
		{/snippet}
	</ProfileCard>

	<SettingsSection
		title="Identité"
		description="Le nom et l’adresse que voient les autres membres de vos calendriers."
	>
		<SettingsRow
			label="Nom"
			description="Affiché sur les événements que vous créez et dans les listes de membres."
			for="profile-name"
			stacked
		>
			<Input id="profile-name" value={app.user?.name ?? ''} disabled />
		</SettingsRow>

		<SettingsRow
			label="Email"
			description="Votre identifiant CalDAV et l’adresse par laquelle on vous partage un calendrier."
			for="profile-email"
			stacked
		>
			<Input id="profile-email" type="email" value={app.user?.email ?? ''} disabled />
		</SettingsRow>

		{#if managedBySSO}
			<SettingsRow
				label="Photo de profil"
				description="Elle vient de votre fournisseur SSO. Changez-la dans Porte, elle se met à jour ici en quelques minutes."
			>
				<Badge tone="info">Gérée par le SSO</Badge>
			</SettingsRow>
		{/if}
	</SettingsSection>

	<SettingsSection
		title="Authentification"
		description="Comment cette instance vous laisse entrer. C’est le serveur qui le décide, pas vous."
	>
		{#if ssoOnly}
			<SettingsRow
				label="Authentification unique seule"
				description="Les mots de passe sont désactivés sur cette instance. Vos identifiants appartiennent à votre fournisseur d’identité — changez-les là-bas."
			>
				<Badge tone="info">Gérée par votre fournisseur d’identité</Badge>
			</SettingsRow>
		{:else}
			<SettingsRow
				label="Mot de passe"
				description="Agenda ne réaffiche jamais un mot de passe. Il sert aussi de mot de passe CalDAV pour les comptes locaux."
			>
				<Badge tone="neutral">Compte local</Badge>
			</SettingsRow>

			{#if oidcEnabled}
				<SettingsRow
					label="Authentification unique"
					description="Cette instance accepte aussi le SSO. S’y connecter rejoint le même compte tant que le sub OIDC correspond."
				>
					<Badge tone="neutral">Disponible à la connexion</Badge>
				</SettingsRow>
			{/if}
		{/if}
	</SettingsSection>

	<SettingsSection
		title="Session"
		description="Les sessions sont par navigateur. Vous déconnecter ici laisse vos autres appareils tranquilles."
	>
		<SettingsRow
			label="Se déconnecter"
			description="Termine cette session et renvoie vers la page de connexion. Le token API continue de fonctionner — révoquez-le depuis l’onglet API."
		>
			<Button variant="outline" icon={icons.logout} disabled={loggingOut} onclick={logout}>
				{loggingOut ? 'Déconnexion…' : 'Se déconnecter'}
			</Button>
		</SettingsRow>
	</SettingsSection>
</div>
