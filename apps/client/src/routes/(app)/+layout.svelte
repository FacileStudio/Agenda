<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { MobileNav, SideBar, SpaceSwitcher, Topbar, icons } from '@facile/muse';
	import { backend, type UserProfile, type CalendarItem, type SpaceItem } from '$lib/backend';
	import { getSpaceContext, setSpaceContext, spaceId } from '$lib/space-context.svelte';

	let { children } = $props();

	let user = $state<UserProfile | null>(null);
	let loaded = $state(false);
	let calendars = $state<CalendarItem[]>([]);
	let spaces = $state<SpaceItem[]>([]);
	let collapsed = $state(false);

	function setUser(nextUser: UserProfile) {
		user = nextUser;
	}

	setContext('app', {
		get user() { return user; },
		get calendars() { return calendars; },
		setUser,
		refreshCalendars
	});

	async function refreshCalendars() {
		try {
			calendars = await backend.listCalendars(spaceId() ?? undefined);
		} catch {
			calendars = [];
		}
	}

	$effect(() => {
		const _space = spaceId();
		refreshCalendars();
	});

	onMount(async () => {
		try {
			const result = await backend.me();
			user = result.user;
			loaded = true;
			backend.syncProfile().then(async (r) => {
				if (r.synced) {
					const fresh = await backend.me();
					user = fresh.user;
				}
			}).catch(() => {});
			backend.listSpaces().then((list) => { spaces = list; }).catch(() => { spaces = []; });
			await refreshCalendars();
		} catch {
			goto('/login');
		}
	});

	function isActive(href: string) {
		return page.url.pathname === href || page.url.pathname.startsWith(href + '/');
	}

	const navPages = $derived([
		{ label: 'Calendrier', href: '/calendar', icon: icons.calendar, active: isActive('/calendar') },
		{ label: 'Espaces', href: '/spaces', icon: icons.usersGroup, active: isActive('/spaces') }
	]);

	const onSettings = $derived(isActive('/settings') || isActive('/profile'));

	/**
	 * The rail is 220px wide, so it wants a name and not an address — the local part of
	 * the email stands in when the account has no display name. An empty `avatar_url`
	 * has to become `undefined` or the card renders a broken image instead of initials.
	 */
	const navUser = $derived({
		name: user?.name?.trim() || user?.email?.split('@')[0] || 'Mon profil',
		avatar: user?.avatar_url || undefined
	});

	const switcherSpaces = $derived(spaces.map((s) => ({ id: String(s.id), name: s.name })));

	const activeSpaceId = $derived.by(() => {
		const current = getSpaceContext();
		return current === 'personal' ? null : String(current.spaceId);
	});

	function selectSpace(id: string | null) {
		if (id === null) {
			setSpaceContext('personal');
			return;
		}
		const space = spaces.find((s) => String(s.id) === id);
		if (space) setSpaceContext({ spaceId: space.id, name: space.name, role: space.role });
	}
</script>

{#if loaded}
	<div class="flex h-[100dvh] w-full overflow-hidden bg-fc-page">
		<div class="hidden h-full shrink-0 p-3 md:block">
			<SideBar
				class="h-full"
				icon="solar:calendar-bold-duotone"
				title="Agenda"
				bind:collapsed
				pages={navPages}
				user={navUser}
				userHref="/settings"
				userActive={onSettings}
				spaces={switcherSpaces}
				{activeSpaceId}
				onSpaceSelect={selectSpace}
				personalSpaceLabel="Personnel"
				manageSpacesHref="/spaces"
				manageSpacesLabel="Gérer les espaces"
			/>
		</div>
		<main class="flex min-w-0 flex-1 flex-col overflow-hidden">
			{#if switcherSpaces.length > 0}
				<Topbar class="md:hidden">
					<span class="text-fc-md font-semibold text-fc-fg">Agenda</span>
					<div class="min-w-0 max-w-56 flex-1">
						<SpaceSwitcher
							spaces={switcherSpaces}
							activeId={activeSpaceId}
							onSelect={selectSpace}
							personalLabel="Personnel"
							manageHref="/spaces"
							manageLabel="Gérer les espaces"
						/>
					</div>
				</Topbar>
			{/if}
			<div class="min-h-0 flex-1 overflow-auto overscroll-contain pb-24 md:pb-0">
				{@render children()}
			</div>
		</main>
		<MobileNav
			items={navPages.map(({ href, label, icon, active }) => ({ href, label, icon, active }))}
			user={navUser}
			profileHref="/settings"
			profileActive={onSettings}
		/>
	</div>
{/if}
