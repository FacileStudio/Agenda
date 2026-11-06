<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import UserMenu from '$lib/components/UserMenu.svelte';
	import { ApiError, backend, type UserProfile, type CalendarItem } from '$lib/backend';
	import { sortCalendars } from '$lib/calendar-filter.svelte';
	import { setSessionLocale, t } from '$lib/i18n/index.svelte';

	let { children } = $props();

	let user = $state<UserProfile | null>(null);
	let loaded = $state(false);
	let calendars = $state<CalendarItem[]>([]);

	function setUser(nextUser: UserProfile) {
		user = nextUser;
		setSessionLocale(nextUser.locale);
	}

	setContext('app', {
		get user() { return user; },
		get calendars() { return calendars; },
		get navUser() { return navUser; },
		setUser,
		refreshCalendars
	});

	async function refreshCalendars() {
		try {
			calendars = sortCalendars(await backend.listCalendars());
		} catch {
			calendars = [];
		}
	}

	onMount(async () => {
		try {
			const result = await backend.me();
			setUser(result.user);
			loaded = true;
			backend.syncProfile().then(async (r) => {
				if (r.synced) {
					const fresh = await backend.me();
					setUser(fresh.user);
				}
			}).catch(() => {});
			await refreshCalendars();
		} catch (err) {
			if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
				goto('/login');
			}
		}
	});

	const navUser = $derived({
		name: user?.name?.trim() || user?.email?.split('@')[0] || t('menu.myProfile'),
		email: user?.email ?? '',
		avatar: user?.avatar_url || undefined
	});
</script>

{#if loaded}
	<div class="flex h-[100dvh] w-full flex-col overflow-hidden bg-fc-page">
		{#if !page.url.pathname.startsWith('/calendar')}
			<div class="flex shrink-0 items-center px-3 pt-3 sm:px-5">
				<UserMenu />
			</div>
		{/if}
		<main class="min-h-0 flex-1 overflow-auto overscroll-contain">
			{@render children()}
		</main>
	</div>
{/if}
