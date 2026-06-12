<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend, type UserProfile, type CalendarItem } from '$lib/backend';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import MobileNav from '$lib/components/MobileNav.svelte';

	let { children } = $props();

	let user = $state<UserProfile | null>(null);
	let loaded = $state(false);
	let calendars = $state<CalendarItem[]>([]);

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
			calendars = await backend.listCalendars();
		} catch {
			calendars = [];
		}
	}

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
			await refreshCalendars();
		} catch {
			goto('/login');
		}
	});
</script>

{#if loaded}
	<div class="flex h-screen w-full overflow-hidden">
		<Sidebar {user} {calendars} />
		<main class="flex-1 overflow-auto pb-24 md:pb-0">
			{@render children()}
		</main>
		<MobileNav />
	</div>
{/if}
