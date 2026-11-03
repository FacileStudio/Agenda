<script lang="ts">
	import { goto } from '$app/navigation';
	import { Avatar, icons } from '@facile/muse';
	import { toast } from 'svelte-sonner';
	import { backend } from '$lib/backend';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';

	let { name, email, avatar }: { name: string; email: string; avatar?: string } = $props();

	const links = [
		{ label: 'Calendrier', href: '/calendar', icon: icons.calendar },
		{ label: 'Paramètres', href: '/settings', icon: icons.settings },
		{ label: 'Espaces', href: '/spaces', icon: icons.space },
		{ label: 'Calendriers et membres', href: '/settings/members', icon: icons.usersGroup }
	];

	async function logout() {
		try {
			await backend.logout();
		} catch {
			toast.warning('Le serveur n’a pas confirmé la déconnexion, la session est fermée ici.');
		}
		await goto('/login');
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class="shrink-0 cursor-pointer rounded-fc-pill focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
		aria-label="Menu du profil"
	>
		<Avatar src={avatar} {name} size="sm" />
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="w-60">
		<DropdownMenu.Label class="flex flex-col gap-0.5">
			<span class="truncate text-fc-sm font-semibold text-fc-fg">{name}</span>
			<span class="truncate text-fc-xs font-normal text-fc-fg-muted">{email}</span>
		</DropdownMenu.Label>
		<DropdownMenu.Separator />
		{#each links as link (link.href)}
			<DropdownMenu.Item onSelect={() => goto(link.href)}>
				<iconify-icon icon={link.icon} width="16" height="16" class="block size-4"></iconify-icon>
				{link.label}
			</DropdownMenu.Item>
		{/each}
		<DropdownMenu.Separator />
		<DropdownMenu.Item variant="destructive" onSelect={logout}>
			<iconify-icon icon={icons.logout} width="16" height="16" class="block size-4"></iconify-icon>
			Se déconnecter
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>
