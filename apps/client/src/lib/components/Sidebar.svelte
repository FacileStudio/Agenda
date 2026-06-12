<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getContext } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Separator } from '$lib/components/ui/separator';
	import { backend, type UserProfile, type CalendarItem } from '$lib/backend';
	import CreateCalendarModal from './CreateCalendarModal.svelte';
	import ManageCalendarModal from './ManageCalendarModal.svelte';

	let { user, calendars }: { user: UserProfile | null; calendars: CalendarItem[] } = $props();

	const app = getContext<{ refreshCalendars: () => Promise<void> }>('app');

	let createOpen = $state(false);
	let managedCalendar = $state<CalendarItem | null>(null);
	let manageOpen = $state(false);
	let avatarFailed = $state(false);

	// Reset the fallback when the avatar URL changes (e.g. after profile sync).
	$effect(() => {
		void user?.avatar_url;
		avatarFailed = false;
	});

	function openManage(cal: CalendarItem) {
		managedCalendar = cal;
		manageOpen = true;
	}

	function getInitials(value: string) {
		const parts = value.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase();
	}

	function userLabel(u: UserProfile | null) {
		return u?.name?.trim() || u?.email || '';
	}

	async function handleLogout() {
		try { await backend.logout(); } catch {}
		goto('/login');
	}

	const navItems = [
		{ href: '/calendar', label: 'Calendrier', icon: 'solar:calendar-linear' },
	];

	const ownedCalendars = $derived(calendars.filter(c => c.role === 'owner'));
	const sharedCalendars = $derived(calendars.filter(c => c.role !== 'owner'));
</script>

<aside class="sticky top-0 flex h-screen w-60 flex-col border-r bg-background">
	<!-- Logo -->
	<div class="flex items-center gap-3 px-5 pt-8 pb-4">
		<iconify-icon icon="solar:calendar-bold-duotone" width="28" class="text-foreground"></iconify-icon>
		<span class="text-2xl font-bold tracking-tight">Agenda</span>
	</div>

	<!-- Navigation -->
	<nav class="flex flex-col gap-1 px-3 pb-2">
		{#each navItems as item}
			{@const active = $page.url.pathname.startsWith(item.href)}
			<a
				href={item.href}
				class="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors {active
					? 'bg-foreground text-background font-medium'
					: 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
			>
				<iconify-icon icon={item.icon} width="16" class="shrink-0"></iconify-icon>
				<span>{item.label}</span>
			</a>
		{/each}
	</nav>

	<!-- Calendars -->
	<div class="flex flex-1 flex-col overflow-hidden px-3 pt-3">
		<div class="mb-2 flex items-center justify-between px-1">
			<p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">Mes calendriers</p>
			<button
				onclick={() => (createOpen = true)}
				class="cursor-pointer rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
				title="Nouveau calendrier"
			>
				<iconify-icon icon="mdi:plus" width="16"></iconify-icon>
			</button>
		</div>

		<div class="flex-1 overflow-y-auto">
			{#each ownedCalendars as cal (cal.id)}
				<div class="group flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-muted">
					<span class="size-2.5 shrink-0 rounded-full" style="background-color: {cal.color}"></span>
					<span class="flex-1 truncate">{cal.name}</span>
					<button
						onclick={() => openManage(cal)}
						class="cursor-pointer shrink-0 rounded-md p-0.5 text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100"
						title="Gérer"
					>
						<iconify-icon icon="solar:settings-minimalistic-linear" width="13"></iconify-icon>
					</button>
				</div>
			{/each}

			{#if sharedCalendars.length > 0}
				<p class="mt-3 mb-1 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Partagés</p>
				{#each sharedCalendars as cal (cal.id)}
					<div class="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm">
						<span class="size-2.5 shrink-0 rounded-full" style="background-color: {cal.color}"></span>
						<span class="flex-1 truncate">{cal.name}</span>
						<span class="text-xs text-muted-foreground">{cal.role === 'writer' || cal.role === 'admin' ? 'Édit.' : 'Lect.'}</span>
					</div>
				{/each}
			{/if}
		</div>
	</div>

	<Separator />

	<!-- Profile section -->
	<div class="flex flex-col gap-2 p-4">
		<a
			href="/settings"
			class="flex cursor-pointer items-center gap-3 rounded-xl border border-border/70 bg-muted/40 p-2.5 transition-colors hover:bg-muted"
		>
			{#if user?.avatar_url && !avatarFailed}
				<img
					src={user.avatar_url}
					alt={userLabel(user)}
					class="h-10 w-10 shrink-0 rounded-full border border-border object-cover"
					onerror={() => (avatarFailed = true)}
				/>
			{:else}
				<div
					class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-border bg-foreground text-sm font-semibold text-background"
				>
					{getInitials(userLabel(user))}
				</div>
			{/if}
			<div class="min-w-0 flex-1">
				<p class="truncate text-sm font-medium">{user?.name || 'Mon profil'}</p>
				<p class="truncate text-xs text-muted-foreground">{user?.email}</p>
			</div>
		</a>
		<Button
			variant="ghost"
			size="sm"
			class="w-full cursor-pointer justify-start gap-2 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
			onclick={handleLogout}
		>
			<iconify-icon icon="solar:logout-2-linear" width="16"></iconify-icon>
			Déconnexion
		</Button>
	</div>
</aside>

<CreateCalendarModal
	bind:open={createOpen}
	onCreated={() => { createOpen = false; app.refreshCalendars(); }}
	onClose={() => (createOpen = false)}
/>

<ManageCalendarModal
	bind:open={manageOpen}
	calendar={managedCalendar}
	onUpdated={() => { app.refreshCalendars(); }}
	onDeleted={() => { manageOpen = false; managedCalendar = null; app.refreshCalendars(); }}
	onClose={() => { manageOpen = false; managedCalendar = null; }}
/>
