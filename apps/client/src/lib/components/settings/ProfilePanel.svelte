<script lang="ts">
	import { getContext } from 'svelte';
	import { Button, ProfileSection, icons, toast } from '@facile/muse';
	import { backend, type UserProfile } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	const app = getContext<{ user: UserProfile | null; setUser: (u: UserProfile) => void }>('app');

	let syncing = $state(false);

	const displayName = $derived(app.user?.name?.trim() || app.user?.email || '');

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
			toast.danger(t('profile.syncFailed'));
		} finally {
			syncing = false;
		}
	}
</script>

<ProfileSection name={displayName} email={app.user?.email} avatar={app.user?.avatar_url || undefined}>
	{#snippet actions()}
		{#if app.user?.avatar_source}
			<Button variant="outline" icon={icons.refresh} disabled={syncing} onclick={syncProfile}>
				{syncing ? t('profile.syncing') : t('profile.sync')}
			</Button>
		{/if}
	{/snippet}
</ProfileSection>
