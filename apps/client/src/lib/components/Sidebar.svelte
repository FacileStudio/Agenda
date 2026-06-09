<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { backend, type UserProfile, type CalendarItem } from '$lib/backend';

	let { user, calendars }: { user: UserProfile | null; calendars: CalendarItem[] } = $props();

	async function handleLogout() {
		await backend.logout();
		goto('/login');
	}
</script>

<aside class="flex h-full w-64 flex-col border-r bg-background">
	<div class="flex h-14 items-center border-b px-4 font-semibold">
		Agenda
	</div>

	<nav class="flex-1 overflow-y-auto p-2">
		<a
			href="/calendar"
			class="flex items-center gap-2 rounded-md px-3 py-2 text-sm hover:bg-accent
				{$page.url.pathname.startsWith('/calendar') ? 'bg-accent font-medium' : ''}"
		>
			Calendar
		</a>
	</nav>

	<div class="border-t p-3">
		<p class="mb-1 text-xs font-medium text-muted-foreground uppercase tracking-wide">My calendars</p>
		{#each calendars as cal}
			<div class="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm">
				<span class="size-2.5 rounded-full flex-shrink-0" style="background-color: {cal.color}"></span>
				<span class="truncate">{cal.name}</span>
				{#if cal.role !== 'owner'}
					<span class="ml-auto text-xs text-muted-foreground">{cal.role}</span>
				{/if}
			</div>
		{/each}
	</div>

	<div class="border-t p-3">
		{#if user}
			<div class="flex items-center justify-between gap-2">
				<div class="min-w-0">
					<p class="truncate text-sm font-medium">{user.name || user.email}</p>
					<p class="truncate text-xs text-muted-foreground">{user.email}</p>
				</div>
				<button
					onclick={handleLogout}
					class="shrink-0 rounded-md p-1.5 text-xs text-muted-foreground hover:bg-accent"
				>
					Sign out
				</button>
			</div>
		{/if}
	</div>
</aside>
