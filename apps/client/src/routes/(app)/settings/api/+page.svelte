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
	import { t } from '$lib/i18n/index.svelte';
	import { formatDate } from '$lib/i18n/format';

	const app = getContext<{ user: UserProfile | null }>('app');

	let apiToken = $state<ApiTokenStatus | null>(null);
	let tokenBusy = $state(false);
	let createOpen = $state(false);
	let revokeOpen = $state(false);
	let tokenName = $state('CalDAV');
	let issuedToken = $state('');

	const serverOrigin = $derived(backend.baseUrl || (browser ? window.location.origin : ''));
	const caldavPath = $derived(`/dav/${app.user?.email ?? ''}`);
	const caldavUrl = $derived(`${serverOrigin}${caldavPath}`);
	const serverHost = $derived(serverOrigin.replace(/^https?:\/\//, ''));
	const clients = $derived([
		{
			id: 'ios',
			label: t('api.client.ios'),
			icon: 'solar:iphone-linear',
			steps: [
				t('api.step.ios1'),
				t('api.step.ios2'),
				t('api.step.server', { value: serverHost }),
				t('api.step.user'),
				t('api.step.password')
			]
		},
		{
			id: 'macos',
			label: t('api.client.macos'),
			icon: 'solar:monitor-linear',
			steps: [
				t('api.step.macos1'),
				t('api.step.user'),
				t('api.step.password'),
				t('api.step.serverAddress', { value: serverHost }),
				t('api.step.macosPath', { value: caldavPath }),
				t('api.step.macosPort')
			]
		},
		{
			id: 'davx5',
			label: t('api.client.davx5'),
			icon: 'solar:smartphone-linear',
			steps: [
				t('api.step.davx51'),
				t('api.step.baseUrl', { value: caldavUrl }),
				t('api.step.user'),
				t('api.step.password')
			]
		},
		{
			id: 'thunderbird',
			label: t('api.client.thunderbird'),
			icon: 'solar:letter-linear',
			steps: [
				t('api.step.tb1'),
				t('api.step.location', { value: caldavUrl }),
				t('api.step.tbUser'),
				t('api.step.password')
			]
		}
	]);

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
			const wasReplacing = Boolean(apiToken?.has_token);
			const created = await backend.createApiToken(tokenName || 'CalDAV');
			issuedToken = created.token;
			apiToken = { has_token: true, name: created.name, created_at: created.created_at };
			toast.success(wasReplacing ? t('api.toastReplaced') : t('api.toastGenerated'));
		} catch (error: unknown) {
			toast.error(error instanceof Error ? error.message : t('common.error'));
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
			toast.success(t('api.toastRevoked'));
		} catch (error: unknown) {
			toast.error(error instanceof Error ? error.message : t('common.error'));
		} finally {
			tokenBusy = false;
		}
	}

	function formatDay(value?: string) {
		return value ? formatDate(value) : '—';
	}
</script>

<svelte:head>
	<title>{t('settings.tab.api')} — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<SettingsSection
		title={t('api.caldav')}
		description={t('api.caldavText')}
	>
		<SettingsRow label={t('api.server')} description={t('api.serverText')} stacked>
			<SecretField value={serverOrigin} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow label={t('api.path')} description={t('api.pathText')} stacked>
			<SecretField value={caldavPath} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow label={t('api.fullUrl')} description={t('api.fullUrlText')} stacked>
			<SecretField value={caldavUrl} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow label={t('api.username')} description={t('api.usernameText')} stacked>
			<SecretField value={app.user?.email ?? ''} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow label={t('api.password')} description={t('api.passwordText')}>
			<StatusDot tone="warning" label={t('api.tokenRequired')} />
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('api.token')} description={t('api.tokenText')}>
		{#snippet actions()}
			<Button icon={icons.plus} onclick={openCreate}>
				{apiToken?.has_token ? t('api.tokenReplace') : t('api.tokenGenerate')}
			</Button>
		{/snippet}

		{#if !apiToken?.has_token}
			<Alert tone="warning">{t('api.tokenAlert')}</Alert>
		{/if}

		{#if apiToken?.has_token}
			<Table>
				<thead>
					<tr>
						<th>{t('api.colName')}</th>
						<th>{t('api.colCreated')}</th>
						<th>{t('api.colScope')}</th>
						<th class="text-right">{t('api.colAction')}</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td class="font-fc-mono">{apiToken.name ?? 'Token'}</td>
						<td>{formatDay(apiToken.created_at)}</td>
						<td>{t('api.scope')}</td>
						<td class="text-right">
							<Button
								variant="ghost-danger"
								size="sm"
								icon={icons.revoke}
								disabled={tokenBusy}
								onclick={() => (revokeOpen = true)}
							>
								{t('common.revoke')}
							</Button>
						</td>
					</tr>
				</tbody>
			</Table>
		{:else}
			<SettingsRow label={t('api.noToken')} description={t('api.noTokenText')}>
				<StatusDot tone="neutral" label={t('api.notGenerated')} />
			</SettingsRow>
		{/if}
	</SettingsSection>

	<SettingsSection
		title={t('api.instructions')}
		description={t('api.instructionsText')}
		bare
	>
		<div class="overflow-hidden rounded-fc-md bg-fc-component">
			{#each clients as client (client.id)}
				<details class="group [&+details]:mt-px">
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
						{#each client.steps as step (step)}
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
	title={apiToken?.has_token && !issuedToken ? t('api.drawerReplaceTitle') : t('api.drawerGenerateTitle')}
	description={apiToken?.has_token && !issuedToken
		? t('api.drawerReplaceText')
		: t('api.drawerGenerateText')}
	onClose={() => (issuedToken = '')}
>
	{#if issuedToken}
		<div class="flex flex-col gap-4">
			<Alert tone="warning" title={t('api.copyNow')}>{t('api.copyNowText')}</Alert>
			<SecretField
				value={issuedToken}
				label={t('api.tokenLabel')}
				helper={t('api.tokenHelper')}
				visible
				autoHideMs={0}
			/>
		</div>
	{:else}
		<div class="flex flex-col gap-2">
			<label for="token-name" class="text-fc-sm font-medium text-fc-fg">{t('api.tokenName')}</label>
			<Input id="token-name" bind:value={tokenName} placeholder={t('api.tokenNamePlaceholder')} />
			<p class="text-fc-xs text-fc-fg-muted">{t('api.tokenScopeNote')}</p>
		</div>
	{/if}

	{#snippet footer()}
		{#if issuedToken}
			<Button variant="outline" onclick={() => (createOpen = false)}>{t('common.close')}</Button>
		{:else}
			<Button variant="outline" onclick={() => (createOpen = false)}>{t('common.cancel')}</Button>
			<Button icon={icons.key} disabled={tokenBusy} onclick={generateToken}>
				{tokenBusy ? t('api.generating') : apiToken?.has_token ? t('api.replaceButton') : t('api.generateButton')}
			</Button>
		{/if}
	{/snippet}
</Drawer>

<ConfirmModal
	bind:open={revokeOpen}
	title={t('api.revokeTitle')}
	description={t('api.revokeText')}
	confirmLabel={t('common.revoke')}
	tone="danger"
	icon={icons.revoke}
	onConfirm={revokeToken}
/>
