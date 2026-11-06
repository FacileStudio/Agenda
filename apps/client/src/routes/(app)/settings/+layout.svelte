<script lang="ts">
	/**
	 * CHARTE §14: one settings page, top pill tabs, then a rule.
	 *
	 * The section lives in the URL rather than in a local `$state`, so `/settings/api`
	 * deep-links, a reload stays put and browser-back walks the sections — the whole
	 * reason `Tabs` items carry `href`. The `gap-4` under the strip is the header/body
	 * separation: pulled tighter the rule reads as an underline welded to the pill.
	 */
	import { page } from '$app/state';
	import { Divider, PageTransition, Tabs, icons } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';

	let { children } = $props();

	const sections = $derived([
		{ id: 'profile', label: t('settings.tab.profile'), icon: icons.userCircle, href: '/settings' },
		{
			id: 'appearance',
			label: t('settings.tab.appearance'),
			icon: icons.palette,
			href: '/settings/appearance'
		},
		{
			id: 'notifications',
			label: t('settings.tab.notifications'),
			icon: icons.notification,
			href: '/settings/notifications'
		},
		{ id: 'api', label: t('settings.tab.api'), icon: icons.key, href: '/settings/api' },
		{
			id: 'members',
			label: t('settings.tab.members'),
			icon: icons.usersGroup,
			href: '/settings/members'
		},
		{ id: 'advanced', label: t('settings.tab.advanced'), icon: icons.shield, href: '/settings/advanced' }
	]);

	const active = $derived(
		sections.find((section) => section.href === page.url.pathname)?.id ?? 'profile'
	);
</script>

<div class="mx-auto flex w-full max-w-fc-lg flex-col gap-8 px-4 py-6 md:px-8 md:py-10">
	<div class="flex flex-col gap-1">
		<h1 class="text-fc-2xl font-semibold text-fc-fg">{t('settings.title')}</h1>
		<p class="text-fc-sm text-fc-fg-muted">
			{t('settings.subtitle')}
		</p>
	</div>

	<div class="flex flex-col gap-4">
		<Tabs items={sections} value={active} label={t('settings.sections')} />
		<Divider class="my-0" />
	</div>

	<PageTransition key={active} distance={8} duration={0.25}>
		{@render children?.()}
	</PageTransition>
</div>
