<script lang="ts">
	/**
	 * CHARTE §14: every credential goes through `SecretField` — no hand-rolled <code> with
	 * an eye button. The endpoint, the path and the login are machine strings rather than
	 * secrets, so they run with `sensitive={false}`: copy, no mask.
	 *
	 * The token is issued in a `Drawer` whose body swaps to the revealed value on success,
	 * and reopening resets that state so a previous token can never reappear.
	 */
	import { getContext, onMount } from 'svelte';
	import { browser } from '$app/environment';
	import {
		Alert,
		Button,
		ConfirmModal,
		Drawer,
		Input,
		SecretField,
		SettingsRow,
		SettingsSection,
		StatusDot,
		Table,
		icons
	} from '@facile/muse';
	import { toast } from 'svelte-sonner';
	import { backend, type ApiTokenStatus, type UserProfile } from '$lib/backend';

	const app = getContext<{ user: UserProfile | null }>('app');

	let apiToken = $state<ApiTokenStatus | null>(null);
	let tokenBusy = $state(false);
	let createOpen = $state(false);
	let revokeOpen = $state(false);
	let tokenName = $state('CalDAV');
	let issuedToken = $state('');

	const isSSOUser = $derived(app.user?.avatar_source === 'oidc');
	const serverOrigin = $derived(backend.baseUrl || (browser ? window.location.origin : ''));
	const caldavPath = $derived(`/dav/${app.user?.email ?? ''}`);
	const caldavUrl = $derived(`${serverOrigin}${caldavPath}`);
	const serverHost = $derived(serverOrigin.replace(/^https?:\/\//, ''));
	const passwordHint = $derived(isSSOUser ? 'le token API ci-dessus' : 'votre mot de passe');

	const clients = [
		{
			id: 'ios',
			label: 'iOS — Calendrier',
			icon: 'solar:iphone-linear',
			steps: () => [
				'Réglages → Calendrier → Comptes → Ajouter un compte → Autre',
				'Ajouter un compte CalDAV',
				`Serveur : ${serverHost}`,
				'Nom d’utilisateur : votre email',
				`Mot de passe : ${passwordHint}`
			]
		},
		{
			id: 'macos',
			label: 'macOS — Calendrier',
			icon: 'solar:monitor-linear',
			steps: () => [
				'Calendrier → Réglages → Comptes → + → CalDAV (Avancé)',
				'Nom d’utilisateur : votre email',
				`Mot de passe : ${passwordHint}`,
				`Adresse du serveur : ${serverHost}`,
				`Chemin d’accès : ${caldavPath}`,
				'Port : 443 — SSL activé'
			]
		},
		{
			id: 'davx5',
			label: 'Android — DAVx⁵',
			icon: 'solar:smartphone-linear',
			steps: () => [
				'DAVx⁵ → + → Connexion avec URL et nom d’utilisateur',
				`URL de base : ${caldavUrl}`,
				'Nom d’utilisateur : votre email',
				`Mot de passe : ${passwordHint}`
			]
		},
		{
			id: 'thunderbird',
			label: 'Thunderbird',
			icon: 'solar:letter-linear',
			steps: () => [
				'Agenda → Nouveau calendrier → Sur le réseau → CalDAV',
				`Emplacement : ${caldavUrl}`,
				'Identifiant : votre email',
				`Mot de passe : ${passwordHint}`
			]
		}
	];

	onMount(async () => {
		try {
			apiToken = await backend.getApiToken();
		} catch {
			apiToken = null;
		}
	});

	function openCreate() {
		issuedToken = '';
		tokenName = 'CalDAV';
		createOpen = true;
	}

	async function generateToken() {
		tokenBusy = true;
		try {
			const created = await backend.createApiToken(tokenName || 'CalDAV');
			issuedToken = created.token;
			apiToken = { has_token: true, name: created.name, created_at: created.created_at };
		} catch (error: unknown) {
			toast.error(error instanceof Error ? error.message : 'Erreur.');
		} finally {
			tokenBusy = false;
		}
	}

	async function revokeToken() {
		tokenBusy = true;
		try {
			await backend.deleteApiToken();
			apiToken = { has_token: false };
			issuedToken = '';
			toast.success('Token révoqué.');
		} catch (error: unknown) {
			toast.error(error instanceof Error ? error.message : 'Erreur.');
		} finally {
			tokenBusy = false;
		}
	}

	function formatDate(value?: string) {
		return value ? new Date(value).toLocaleDateString('fr-FR') : '—';
	}
</script>

<svelte:head>
	<title>API — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<SettingsSection
		title="Connexion CalDAV"
		description="Synchronisez vos calendriers avec Calendrier iOS, Calendrier macOS, DAVx⁵, Thunderbird ou tout autre client CalDAV."
	>
		<SettingsRow
			label="Serveur"
			description="L’hôte à saisir quand le client demande une adresse plutôt qu’une URL complète."
			stacked
		>
			<SecretField value={serverOrigin} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow
			label="Chemin d’accès"
			description="Requis par Calendrier macOS en mode CalDAV avancé."
			stacked
		>
			<SecretField value={caldavPath} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow
			label="URL complète"
			description="L’URL de base pour DAVx⁵ et Thunderbird — serveur et chemin d’un seul tenant."
			stacked
		>
			<SecretField value={caldavUrl} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow
			label="Identifiant"
			description="Votre adresse email. L’authentification CalDAV est un Basic Auth classique."
			stacked
		>
			<SecretField value={app.user?.email ?? ''} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow
			label="Mot de passe"
			description={isSSOUser
				? 'Vous vous connectez par SSO, vous n’avez donc pas de mot de passe Agenda : le token API ci-dessous en tient lieu.'
				: 'Votre mot de passe Agenda. Il n’est jamais réaffiché — s’il est perdu, générez plutôt un token API.'}
		>
			<StatusDot
				tone={isSSOUser ? 'warning' : 'success'}
				label={isSSOUser ? 'Token API requis' : 'Mot de passe Agenda'}
			/>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title="Token API"
		description="Un identifiant machine pour les clients CalDAV et les appels HTTP. Agenda en garde un seul par compte."
	>
		{#snippet actions()}
			{#if !apiToken?.has_token}
				<Button icon={icons.plus} onclick={openCreate}>Générer un token</Button>
			{/if}
		{/snippet}

		{#if isSSOUser}
			<Alert tone="warning">
				Vous êtes connecté via SSO — vous n’avez pas de mot de passe Agenda. Générez un token API
				et utilisez-le comme mot de passe dans votre client CalDAV.
			</Alert>
		{/if}

		{#if apiToken?.has_token}
			<Table>
				<thead>
					<tr>
						<th>Nom</th>
						<th>Créé le</th>
						<th>Portée</th>
						<th class="text-right">Action</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td class="font-fc-mono">{apiToken.name ?? 'Token'}</td>
						<td>{formatDate(apiToken.created_at)}</td>
						<td>CalDAV et API — votre compte</td>
						<td class="text-right">
							<Button
								variant="ghost-danger"
								size="sm"
								icon={icons.revoke}
								disabled={tokenBusy}
								onclick={() => (revokeOpen = true)}
							>
								Révoquer
							</Button>
						</td>
					</tr>
				</tbody>
			</Table>
		{:else}
			<SettingsRow
				label="Aucun token"
				description="Le token n’est affiché qu’une fois, à sa création. Agenda n’en stocke qu’un hash — il ne peut pas vous le rappeler ensuite."
			>
				<StatusDot tone="neutral" label="Non généré" />
			</SettingsRow>
		{/if}
	</SettingsSection>

	<SettingsSection
		title="Instructions par client"
		description="Les mêmes identifiants, dans l’ordre où chaque application les demande."
		bare
	>
		<div class="overflow-hidden rounded-fc-md bg-fc-component">
			{#each clients as client (client.id)}
				<details class="group border-t border-fc-border first:border-t-0">
					<summary
						class="flex min-h-11 cursor-pointer list-none items-center justify-between gap-3 px-4 py-3 text-fc-sm font-medium text-fc-fg select-none"
					>
						<span class="flex items-center gap-2">
							<iconify-icon icon={client.icon} width="16" class="text-fc-fg-muted"></iconify-icon>
							{client.label}
						</span>
						<iconify-icon
							icon="solar:alt-arrow-down-linear"
							width="14"
							class="text-fc-fg-muted transition-transform group-open:rotate-180"
						></iconify-icon>
					</summary>
					<ol class="list-decimal space-y-1 px-4 pt-1 pb-4 pl-10 text-fc-sm text-fc-fg-muted">
						{#each client.steps() as step (step)}
							<li>{step}</li>
						{/each}
					</ol>
				</details>
			{/each}
		</div>
	</SettingsSection>
</div>

<Drawer
	bind:open={createOpen}
	title="Générer un token API"
	description="Nommez-le d’après la machine qui s’en servira — c’est la seule façon de savoir plus tard ce que vous révoquez."
	onClose={() => (issuedToken = '')}
>
	{#if issuedToken}
		<div class="flex flex-col gap-4">
			<Alert tone="warning" title="Copiez-le maintenant">
				Ce token ne sera plus jamais affiché. Agenda n’en conserve qu’un hash : ferme ce tiroir,
				il est irrécupérable et il faudra en générer un autre.
			</Alert>
			<SecretField
				value={issuedToken}
				label="Token"
				helper="À utiliser comme mot de passe dans votre client CalDAV."
				visible
				autoHideMs={0}
			/>
		</div>
	{:else}
		<div class="flex flex-col gap-2">
			<label for="token-name" class="text-fc-sm font-medium text-fc-fg">Nom du token</label>
			<Input id="token-name" bind:value={tokenName} placeholder="MacBook, iPhone, serveur CI…" />
			<p class="text-fc-xs text-fc-fg-muted">
				Agenda n’attache ni portée ni expiration à un token : il vaut votre compte, pour CalDAV
				comme pour l’API. Révoquez-le dès qu’une machine n’en a plus besoin.
			</p>
		</div>
	{/if}

	{#snippet footer()}
		{#if issuedToken}
			<Button variant="outline" onclick={() => (createOpen = false)}>Fermer</Button>
		{:else}
			<Button variant="outline" onclick={() => (createOpen = false)}>Annuler</Button>
			<Button icon={icons.key} disabled={tokenBusy} onclick={generateToken}>
				{tokenBusy ? 'Génération…' : 'Générer'}
			</Button>
		{/if}
	{/snippet}
</Drawer>

<ConfirmModal
	bind:open={revokeOpen}
	title="Révoquer le token API ?"
	description="Tout client CalDAV et tout script qui s’en sert cesse de se synchroniser immédiatement, et un token révoqué ne se restaure pas — il faudra en générer un nouveau et le ressaisir sur chaque appareil."
	confirmLabel="Révoquer"
	cancelLabel="Annuler"
	tone="danger"
	icon={icons.revoke}
	onConfirm={revokeToken}
/>
