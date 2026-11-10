<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import {
		Button,
		DangerZone,
		SecretField,
		SettingsRow,
		SettingsSection,
		StatusDot,
		icons
	} from '@facile/muse';
	import { backend, registreSpaceUrl } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let oidcEnabled = $state(false);
	let configLoaded = $state(false);
	let encryptionKeySet = $state(false);
	let settingsLoaded = $state(false);

	const serverOrigin = browser ? window.location.origin : '';
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
			<Button variant="secondary" href="/settings/caldav" icon={icons.download}>
				{t('advanced.connect')}
			</Button>
		</SettingsRow>
	</SettingsSection>

	<DangerZone title={t('advanced.danger')} description={t('advanced.dangerAlert')}>
		<Button variant="secondary" href="/settings/members" icon={icons.calendar}>
			{t('advanced.openCalendars')}
		</Button>
		<Button variant="secondary" href="/settings/tokens" icon={icons.key}>{t('advanced.openApi')}</Button>
		<Button
			variant="ghost-danger"
			href={registreSpaceUrl()}
			target="_blank"
			rel="noopener"
			icon={icons.warning}
		>
			{t('advanced.leaveSpace')}
		</Button>
	</DangerZone>
</div>
