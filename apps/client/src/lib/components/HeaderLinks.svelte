<script lang="ts">
	import { Button, Tooltip, icons } from '@facile/muse';
	import { page } from '$app/state';
	import { t } from '$lib/i18n/index.svelte';

	let { class: className = '' }: { class?: string } = $props();

	const links = [{ href: '/calendar', icon: icons.calendar, label: t('menu.calendar') }];
</script>

<div class="flex min-w-0 flex-wrap items-center gap-1 {className}">
	{#each links as link (link.href)}
		{@const active = page.url.pathname.startsWith(link.href)}
		<Tooltip text={link.label} describe={false} class="contents">
			<Button
				href={link.href}
				variant={active ? 'secondary' : 'ghost'}
				size="lg"
				icon={link.icon}
				aria-label={link.label}
				aria-current={active ? 'page' : undefined}
			>
				<span class="hidden sm:inline">{link.label}</span>
			</Button>
		</Tooltip>
	{/each}
</div>
