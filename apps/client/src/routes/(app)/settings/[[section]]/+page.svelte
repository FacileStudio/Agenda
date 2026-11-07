<script lang="ts">
	import { SettingsLayout } from '@facile/muse';
	import AdvancedPanel from '$lib/components/settings/AdvancedPanel.svelte';
	import CaldavPanel from '$lib/components/settings/CaldavPanel.svelte';
	import MembersPanel from '$lib/components/settings/MembersPanel.svelte';
	import NotificationsPanel from '$lib/components/settings/NotificationsPanel.svelte';
	import PreferencesPanel from '$lib/components/settings/PreferencesPanel.svelte';
	import ProfilePanel from '$lib/components/settings/ProfilePanel.svelte';
	import TokensPanel from '$lib/components/settings/TokensPanel.svelte';
	import { SECTION_ICONS } from '$lib/settings-sections';
	import { t } from '$lib/i18n/index.svelte';

	let { data } = $props();

	const sections = $derived.by(() => [
		{ id: 'profile', label: t('settings.tab.profile') },
		{ id: 'preferences', label: t('settings.tab.preferences') },
		{ id: 'notifications', label: t('settings.tab.notifications') },
		{ id: 'caldav', label: t('settings.tab.caldav') },
		{ id: 'members', label: t('settings.tab.members') },
		{ id: 'tokens', label: t('settings.tab.tokens') },
		{ id: 'advanced', label: t('settings.tab.advanced') }
	].map((section) => ({
		...section,
		icon: SECTION_ICONS[section.id as keyof typeof SECTION_ICONS],
		href: `/settings/${section.id}`
	})));
</script>

<svelte:head>
	<title>{t('settings.title')} — Agenda</title>
</svelte:head>

<div class="mx-auto w-full max-w-fc-lg px-4 py-6 md:px-8 md:py-10">
	<SettingsLayout
		{sections}
		active={data.section}
		backHref="/settings"
		title={t('settings.title')}
		description={t('settings.subtitle')}
	>
		{#snippet children(id)}
			{#if id === 'profile'}
				<ProfilePanel />
			{:else if id === 'preferences'}
				<PreferencesPanel />
			{:else if id === 'notifications'}
				<NotificationsPanel />
			{:else if id === 'caldav'}
				<CaldavPanel />
			{:else if id === 'members'}
				<MembersPanel />
			{:else if id === 'tokens'}
				<TokensPanel />
			{:else if id === 'advanced'}
				<AdvancedPanel />
			{/if}
		{/snippet}
	</SettingsLayout>
</div>
