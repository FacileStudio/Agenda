<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { UserMenu, icons, toast } from '@facile/muse';
	import { ApiError, backend, type UserProfile, type CalendarItem } from '$lib/backend';
	import { sortCalendars } from '$lib/calendar-filter.svelte';
	import { rememberLocale, setSessionLocale, t } from '$lib/i18n/index.svelte';
	import { theme, type ThemePreference } from '$lib/theme.svelte';

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
		get userMenu() { return userMenu; },
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

	async function signOut() {
		try {
			await backend.logout();
		} catch {
			toast.warning(t('menu.logoutUnconfirmed'));
		}
		await goto('/login');
	}

	const userMenu = $derived({
		items: [
			{ label: t('menu.calendar'), icon: icons.calendar, href: '/calendar' },
			{ label: t('menu.spaces'), icon: icons.space, href: '/spaces' },
			{ label: t('menu.members'), icon: icons.usersGroup, href: '/settings/members' }
		],
		settingsHref: '/settings',
		subscription: { plan: '', href: 'https://sso.facile.studio/spaces' },
		helpHref: 'https://github.com/FacileStudio/Agenda/tree/main/docs',
		onSignOut: signOut,
		onLocaleChange: rememberLocale,
		theme: theme.preference,
		onThemeChange: (next: ThemePreference) => theme.set(next)
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
				<UserMenu user={navUser} trigger="avatar" {...userMenu} />
			</div>
		{/if}
		<main class="min-h-0 flex-1 overflow-auto overscroll-contain">
			{@render children()}
		</main>
	</div>
{/if}
