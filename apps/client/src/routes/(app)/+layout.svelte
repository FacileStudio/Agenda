<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { SpaceSwitcher, Topbar } from '@facile/muse';
	import UserMenu from '$lib/components/UserMenu.svelte';
	import { ApiError, backend, type UserProfile, type CalendarItem, type SpaceItem } from '$lib/backend';
	import { getSpaceContext, setSpaceContext, spaceId } from '$lib/space-context.svelte';

	let { children } = $props();

	let user = $state<UserProfile | null>(null);
	let loaded = $state(false);
	let calendars = $state<CalendarItem[]>([]);
	let spaces = $state<SpaceItem[]>([]);

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
			backend.listSpaces().then(adoptSpaces).catch(() => { spaces = []; });
			await refreshCalendars();
		} catch (err) {
			if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
				goto('/login');
			}
		}
	});

	const navUser = $derived({
		name: user?.name?.trim() || user?.email?.split('@')[0] || 'Mon profil',
		email: user?.email ?? '',
		avatar: user?.avatar_url || undefined
	});

	const switcherSpaces = $derived(spaces.map((s) => ({ id: String(s.id), name: s.name })));

	const activeSpaceId = $derived.by(() => {
		const current = getSpaceContext();
		return current === 'personal' ? null : String(current.spaceId);
	});

	function adoptSpaces(list: SpaceItem[]) {
		spaces = list;
		const id = spaceId();
		if (id !== null && !list.some((s) => s.id === id)) setSpaceContext('personal');
	}

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
	<div class="flex h-[100dvh] w-full flex-col overflow-hidden bg-fc-page">
		<Topbar class="shrink-0 justify-start bg-fc-page">
			<UserMenu name={navUser.name} email={navUser.email} avatar={navUser.avatar} />
			<div class="min-w-0 max-w-64 flex-1">
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
		<main class="min-h-0 flex-1 overflow-auto overscroll-contain">
			{@render children()}
		</main>
	</div>
{/if}
