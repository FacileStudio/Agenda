<script lang="ts">
	import { getContext } from 'svelte';
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
	import { t } from '$lib/i18n/index.svelte';
	import { formatDate } from '$lib/i18n/format';

	const app = getContext<{ user: UserProfile | null; setUser: (u: UserProfile) => void }>('app');

	let syncing = $state(false);
	let loggingOut = $state(false);

	const displayName = $derived(app.user?.name?.trim() || app.user?.email || '');
	const managedBySSO = $derived(app.user?.avatar_source === 'oidc');
	const createdAt = $derived(app.user?.created_at ? formatDate(app.user.created_at) : '—');

	async function syncProfile() {
		syncing = true;
		try {
			const result = await backend.syncProfile();
			if (result.synced) {
				const fresh = await backend.me();
				app.setUser(fresh.user);
				toast.success(t('profile.synced'));
			} else {
				toast.info(t('profile.upToDate'));
			}
		} catch {
			toast.error(t('profile.syncFailed'));
		} finally {
			syncing = false;
		}
	}

	async function logout() {
		loggingOut = true;
		try {
			await backend.logout();
		} catch {
			toast.warning(t('menu.logoutUnconfirmed'));
		}
		await goto('/login');
	}
</script>

<svelte:head>
	<title>{t('settings.title')} — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<ProfileCard
		name={displayName}
		email={app.user?.email ?? ''}
		avatar={app.user?.avatar_url ?? ''}
		meta={[
			{ label: t('profile.signedInVia'), value: t('profile.facileAccount') },
			{ label: t('profile.createdOn'), value: createdAt }
		]}
	>
		{#snippet actions()}
			{#if app.user?.avatar_source}
				<Button variant="outline" icon={icons.refresh} disabled={syncing} onclick={syncProfile}>
					{syncing ? t('profile.syncing') : t('profile.sync')}
				</Button>
			{/if}
		{/snippet}
	</ProfileCard>

	<SettingsSection title={t('profile.identity')} description={t('profile.identityText')}>
		<SettingsRow
			label={t('common.name')}
			description={t('profile.nameText')}
			for="profile-name"
			stacked
		>
			<Input id="profile-name" value={app.user?.name ?? ''} disabled />
		</SettingsRow>

		<SettingsRow
			label={t('common.email')}
			description={t('profile.emailText')}
			for="profile-email"
			stacked
		>
			<Input id="profile-email" type="email" value={app.user?.email ?? ''} disabled />
		</SettingsRow>

		{#if managedBySSO}
			<SettingsRow label={t('profile.photo')} description={t('profile.photoText')}>
				<Button variant="outline" href="https://sso.facile.studio/account">
					{t('profile.photoEdit')}
				</Button>
			</SettingsRow>
		{/if}
	</SettingsSection>

	<SettingsSection title={t('profile.auth')} description={t('profile.authText')}>
		<SettingsRow label={t('profile.authRow')} description={t('profile.authRowText')}>
			<Badge tone="info">{t('profile.authBadge')}</Badge>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('profile.session')} description={t('profile.sessionText')}>
		<SettingsRow label={t('menu.logout')} description={t('profile.logoutText')}>
			<Button variant="outline" icon={icons.logout} disabled={loggingOut} onclick={logout}>
				{loggingOut ? t('profile.loggingOut') : t('menu.logout')}
			</Button>
		</SettingsRow>
	</SettingsSection>
</div>
