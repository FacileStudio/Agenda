<script lang="ts">
	import { getContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { Avatar, icons } from '@facile/muse';
	import { toast } from 'svelte-sonner';
	import { backend } from '$lib/backend';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { t } from '$lib/i18n/index.svelte';

	const app = getContext<{ navUser: { name: string; email: string; avatar?: string } }>('app');

	const links = [
		{ key: 'menu.calendar', href: '/calendar', icon: icons.calendar },
		{ key: 'menu.settings', href: '/settings', icon: icons.settings },
		{ key: 'menu.spaces', href: '/spaces', icon: icons.space },
		{ key: 'menu.members', href: '/settings/members', icon: icons.usersGroup }
	] as const;

	async function logout() {
		try {
			await backend.logout();
		} catch {
			toast.warning(t('menu.logoutUnconfirmed'));
		}
		await goto('/login');
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class="flex size-11 shrink-0 cursor-pointer items-center justify-center rounded-fc-pill focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
		aria-label={t('menu.profile')}
	>
		<Avatar src={app.navUser.avatar} name={app.navUser.name} class="size-11 align-top text-fc-sm [&>img]:block" />
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="w-60">
		<DropdownMenu.Label class="flex flex-col gap-0.5">
			<span class="truncate text-fc-sm font-semibold text-fc-fg">{app.navUser.name}</span>
			<span class="truncate text-fc-xs font-normal text-fc-fg-muted">{app.navUser.email}</span>
		</DropdownMenu.Label>
		<DropdownMenu.Separator />
		{#each links as link (link.href)}
			<DropdownMenu.Item onSelect={() => goto(link.href)}>
				<iconify-icon icon={link.icon} width="16" height="16" class="block size-4"></iconify-icon>
				{t(link.key)}
			</DropdownMenu.Item>
		{/each}
		<DropdownMenu.Separator />
		<DropdownMenu.Item variant="destructive" onSelect={logout}>
			<iconify-icon icon={icons.logout} width="16" height="16" class="block size-4"></iconify-icon>
			{t('menu.logout')}
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>
