<script lang="ts">
	/**
	 * CHARTE §14: the danger zone is never its own tab — it sits at the bottom of Advanced,
	 * so a destructive action costs a scroll instead of being one mis-click from every
	 * other tab all day.
	 */
	import { getContext, onMount } from 'svelte';
	import { browser } from '$app/environment';
	import {
		Alert,
		Button,
		SecretField,
		SettingsRow,
		SettingsSection,
		StatusDot,
		icons
	} from '@facile/muse';
	import { backend, type UserProfile } from '$lib/backend';
	import { getSpaceContext } from '$lib/space-context.svelte';

	const app = getContext<{ user: UserProfile | null }>('app');

	let ssoOnly = $state(false);
	let oidcEnabled = $state(false);
	let configLoaded = $state(false);
	let encryptionKeySet = $state(false);
	let settingsLoaded = $state(false);

	const space = $derived(getSpaceContext());
	const serverOrigin = $derived(backend.baseUrl || (browser ? window.location.origin : ''));
	const davEndpoint = $derived(`${serverOrigin}/dav/`);

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
		} finally {
			configLoaded = true;
		}

		try {
			const response = await fetch(`${backend.apiBaseUrl}/settings/`, { credentials: 'include' });
			const payload = (await response.json()) as {
				settings: { encryption_key_set: boolean };
			};
			encryptionKeySet = payload.settings.encryption_key_set;
		} catch {
			encryptionKeySet = false;
		} finally {
			settingsLoaded = true;
		}
	});
</script>

<svelte:head>
	<title>Avancé — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<SettingsSection
		title="Instance"
		description="Les faits à citer quand vous ouvrez un ticket sur une installation auto-hébergée."
	>
		<SettingsRow
			label="Interface"
			description="Cette application est le client web de l’API Agenda, servi par le même binaire."
		>
			<StatusDot tone="neutral" label="Agenda web" />
		</SettingsRow>

		<SettingsRow
			label="Connexion par mot de passe"
			description="Coupée pour toute l’instance par SSO_ONLY, qui masque complètement le formulaire de connexion."
		>
			<StatusDot
				tone={configLoaded && ssoOnly ? 'neutral' : 'success'}
				label={configLoaded ? (ssoOnly ? 'Désactivée' : 'Activée') : 'Vérification…'}
				pulse={!configLoaded}
			/>
		</SettingsRow>

		<SettingsRow
			label="Authentification unique"
			description="Fédération OIDC vers le fournisseur d’identité de l’instance. Activée par OIDC_ISSUER côté serveur."
		>
			<StatusDot
				tone={configLoaded && oidcEnabled ? 'success' : 'neutral'}
				label={configLoaded ? (oidcEnabled ? 'Activée' : 'Non configurée') : 'Vérification…'}
				pulse={!configLoaded}
			/>
		</SettingsRow>

		<SettingsRow
			label="Chiffrement des jetons OIDC"
			description="ENCRYPTION_KEY chiffre au repos les jetons rendus par le fournisseur d’identité. Sans elle ils sont stockés en clair."
		>
			<StatusDot
				tone={settingsLoaded && encryptionKeySet ? 'success' : 'warning'}
				label={settingsLoaded ? (encryptionKeySet ? 'Clé définie' : 'Aucune clé') : 'Vérification…'}
				pulse={!settingsLoaded}
			/>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title="Interopérabilité"
		description="Ce que d’autres logiciels peuvent lire sans passer par cette interface."
	>
		<SettingsRow
			label="Point d’accès CalDAV"
			description="Racine du serveur CalDAV. Les identifiants et le détail par client sont dans l’onglet API."
			stacked
		>
			<SecretField value={davEndpoint} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow
			label="Découverte"
			description="Les clients qui savent le faire trouvent le serveur tout seuls à partir du domaine."
			stacked
		>
			<SecretField value={`${serverOrigin}/.well-known/caldav`} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow
			label="Export d’un calendrier"
			description="Un client CalDAV récupère l’ICS d’origine, alarmes et récurrences comprises — Agenda n’a pas d’export séparé à maintenir."
		>
			<Button variant="outline" href="/settings/api" icon={icons.download}>Connecter un client</Button>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title="Zone de danger"
		description="Les actions irréversibles vivent en bas de cet onglet — jamais dans la barre d’onglets."
	>
		<Alert tone="warning">
			Agenda n’a rien de destructif au niveau du compte : un compte est une identité, il ne
			possède aucune donnée en propre. Ce qui détruit appartient à l’objet concerné — un
			calendrier, un espace, un token — et se fait là où cet objet se gère.
		</Alert>

		<SettingsRow
			label="Supprimer un calendrier"
			description="Emporte ses événements et désabonne tous les clients CalDAV qui le synchronisaient. Réservé au propriétaire, depuis la fiche du calendrier."
		>
			<Button variant="outline" href="/settings/members" icon={icons.calendar}>
				Ouvrir les calendriers
			</Button>
		</SettingsRow>

		<SettingsRow
			label="Révoquer le token API"
			description="Tout client ou script qui s’en sert cesse de se synchroniser immédiatement, et un token révoqué ne se restaure pas."
		>
			<Button variant="outline" href="/settings/api" icon={icons.key}>Ouvrir l’onglet API</Button>
		</SettingsRow>

		<SettingsRow
			label={space === 'personal' ? 'Quitter ou supprimer un espace' : `Quitter « ${space.name} »`}
			description="Quitter un espace vous coupe de ses calendriers partages ; le supprimer les détruit pour tous ses membres."
		>
			<Button
				variant="ghost-danger"
				href={space === 'personal' ? '/spaces' : `/spaces/${space.spaceId}/settings`}
				icon={icons.warning}
			>
				{space === 'personal' ? 'Voir les espaces' : 'Paramètres de l’espace'}
			</Button>
		</SettingsRow>

		{#if app.user}
			<SettingsRow
				label="Supprimer le compte"
				description="Aucun endpoint ne le fait : la suppression d’un compte passe par l’administrateur de l’instance, ou par Porte si vous vous connectez en SSO."
			>
				<StatusDot tone="neutral" label="Via l’administrateur" />
			</SettingsRow>
		{/if}
	</SettingsSection>
</div>
