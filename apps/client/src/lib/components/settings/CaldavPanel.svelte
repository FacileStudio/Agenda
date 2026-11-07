<script lang="ts">
	import { getContext } from 'svelte';
	import { browser } from '$app/environment';
	import { Button, SecretField, SettingsRow, SettingsSection, icons } from '@facile/muse';
	import { backend, type UserProfile } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	const app = getContext<{ user: UserProfile | null }>('app');

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
</script>

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
			<Button variant="outline" href="/settings/tokens" icon={icons.key}>{t('api.tokenRequired')}</Button>
		</SettingsRow>
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
