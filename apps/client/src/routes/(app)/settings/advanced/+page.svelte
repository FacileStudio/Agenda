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
	import { backend, registreSpaceUrl, type UserProfile } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	const app = getContext<{ user: UserProfile | null }>('app');

	let oidcEnabled = $state(false);
	let configLoaded = $state(false);
	let encryptionKeySet = $state(false);
	let settingsLoaded = $state(false);

	const serverOrigin = $derived(backend.baseUrl || (browser ? window.location.origin : ''));
	const davEndpoint = $derived(`${serverOrigin}/dav/`);

	onMount(async () => {
		try {
			const response = await fetch(`${backend.apiBaseUrl}/auth/config`, {
				credentials: 'include'
			});
			const config = (await response.json()) as { oidc_enabled: boolean };
			oidcEnabled = config.oidc_enabled;
		} catch {
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
	<title>{t('settings.tab.advanced')} — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<SettingsSection title={t('advanced.instance')} description={t('advanced.instanceText')}>
		<SettingsRow label={t('advanced.interface')} description={t('advanced.interfaceText')}>
			<StatusDot tone="neutral" label="Agenda web" />
		</SettingsRow>

		<SettingsRow label={t('advanced.sso')} description={t('advanced.ssoText')}>
			<StatusDot
				tone={configLoaded && oidcEnabled ? 'success' : 'neutral'}
				label={configLoaded
					? oidcEnabled
						? t('advanced.enabled')
						: t('advanced.notConfigured')
					: t('advanced.checking')}
				pulse={!configLoaded}
			/>
		</SettingsRow>

		<SettingsRow label={t('advanced.encryption')} description={t('advanced.encryptionText')}>
			<StatusDot
				tone={settingsLoaded && encryptionKeySet ? 'success' : 'warning'}
				label={settingsLoaded
					? encryptionKeySet
						? t('advanced.keySet')
						: t('advanced.keyMissing')
					: t('advanced.checking')}
				pulse={!settingsLoaded}
			/>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('advanced.interop')} description={t('advanced.interopText')}>
		<SettingsRow label={t('advanced.endpoint')} description={t('advanced.endpointText')} stacked>
			<SecretField value={davEndpoint} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow label={t('advanced.discovery')} description={t('advanced.discoveryText')} stacked>
			<SecretField value={`${serverOrigin}/.well-known/caldav`} sensitive={false} class="w-full" />
		</SettingsRow>

		<SettingsRow label={t('advanced.export')} description={t('advanced.exportText')}>
			<Button variant="outline" href="/settings/api" icon={icons.download}>
				{t('advanced.connect')}
			</Button>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('advanced.danger')} description={t('advanced.dangerText')}>
		<Alert tone="warning">{t('advanced.dangerAlert')}</Alert>

		<SettingsRow label={t('advanced.deleteCalendar')} description={t('advanced.deleteCalendarText')}>
			<Button variant="outline" href="/settings/members" icon={icons.calendar}>
				{t('advanced.openCalendars')}
			</Button>
		</SettingsRow>

		<SettingsRow label={t('advanced.revokeToken')} description={t('advanced.revokeTokenText')}>
			<Button variant="outline" href="/settings/api" icon={icons.key}>{t('advanced.openApi')}</Button>
		</SettingsRow>

		<SettingsRow label={t('advanced.leaveSpace')} description={t('advanced.leaveSpaceText')}>
			<Button
				variant="ghost-danger"
				href={registreSpaceUrl()}
				target="_blank"
				rel="noopener"
				icon={icons.warning}
			>
				{t('common.manageInRegistre')}
			</Button>
		</SettingsRow>

		{#if app.user}
			<SettingsRow label={t('advanced.deleteAccount')} description={t('advanced.deleteAccountText')}>
				<StatusDot tone="neutral" label={t('advanced.viaRegistre')} />
			</SettingsRow>
		{/if}
	</SettingsSection>
</div>
