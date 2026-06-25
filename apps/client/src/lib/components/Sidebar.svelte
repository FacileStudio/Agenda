<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getContext } from 'svelte';
	import { backend, type UserProfile, type CalendarItem } from '$lib/backend';
	import CreateCalendarModal from './CreateCalendarModal.svelte';
	import ManageCalendarModal from './ManageCalendarModal.svelte';
	import SpaceSwitcher from './SpaceSwitcher.svelte';

	let { user, calendars }: { user: UserProfile | null; calendars: CalendarItem[] } = $props();

	const app = getContext<{ refreshCalendars: () => Promise<void> }>('app');

	let createOpen = $state(false);
	let managedCalendar = $state<CalendarItem | null>(null);
	let manageOpen = $state(false);
	let avatarFailed = $state(false);

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

	async function logout() {
		try { await backend.logout(); } catch {}
		goto('/login');
	}

	const navLinks = [
		{ href: '/calendar', label: 'Calendrier', icon: 'solar:calendar-linear' },
		{ href: '/spaces', label: 'Espaces', icon: 'solar:users-group-rounded-linear' },
	];

	const ownedCalendars = $derived(calendars.filter(c => c.role === 'owner'));
	const sharedCalendars = $derived(calendars.filter(c => c.role !== 'owner'));
</script>

<aside class="sticky top-0 hidden h-[100dvh] w-60 flex-col border-r bg-background md:flex">
	<div class="flex items-center gap-3 px-5 pt-8 pb-4">
		<iconify-icon icon="solar:calendar-bold-duotone" width="28" class="text-foreground"></iconify-icon>
		<span class="text-2xl font-bold font-heading tracking-tight">Agenda</span>
	</div>

	<SpaceSwitcher />

	<nav class="flex flex-1 flex-col gap-1 px-3">
		{#each navLinks as link}
			{@const active = $page.url.pathname === link.href || $page.url.pathname.startsWith(link.href + '/')}
			<a
				href={link.href}
				class="flex items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors {active
					? 'bg-foreground text-background font-medium'
					: 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
			>
				<iconify-icon icon={link.icon} width="16"></iconify-icon>
				{link.label}
			</a>
		{/each}

		<div class="mt-3 flex flex-col overflow-hidden">
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
					<button
						onclick={() => openManage(cal)}
						class="group flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted"
					>
						<span class="size-2.5 shrink-0 rounded-full" style="background-color: {cal.color}"></span>
						<span class="flex-1 truncate">{cal.name}</span>
						<iconify-icon
							icon="solar:settings-minimalistic-linear"
							width="13"
							class="shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100"
						></iconify-icon>
					</button>
				{/each}

				{#if sharedCalendars.length > 0}
					<p class="mt-3 mb-1 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Partages</p>
					{#each sharedCalendars as cal (cal.id)}
						<button
							onclick={() => openManage(cal)}
							class="flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted"
						>
							<span class="size-2.5 shrink-0 rounded-full" style="background-color: {cal.color}"></span>
							<span class="flex-1 truncate">{cal.name}</span>
							<span class="text-xs text-muted-foreground">{cal.role === 'writer' || cal.role === 'admin' ? 'Edit.' : 'Lect.'}</span>
						</button>
					{/each}
				{/if}
			</div>
		</div>
	</nav>

	<div class="h-px bg-border"></div>

	<div class="flex flex-col gap-2 p-4">
		<a
			href="/settings"
			class="flex items-center gap-3 rounded-xl border p-2.5 transition-colors {$page.url.pathname.startsWith('/settings')
				? 'border-border bg-muted'
				: 'border-border/70 bg-muted/40 hover:bg-muted'}"
		>
			{#if user?.avatar_url && !avatarFailed}
				<img
					src={user.avatar_url}
					alt={userLabel(user)}
					class="h-9 w-9 shrink-0 rounded-full border border-border object-cover"
					onerror={() => (avatarFailed = true)}
				/>
			{:else}
				<div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-border bg-foreground text-xs font-semibold text-background">
					{getInitials(userLabel(user))}
				</div>
			{/if}
			<div class="min-w-0 flex-1">
				<p class="truncate text-sm font-medium">{user?.name || 'Mon profil'}</p>
				<p class="truncate text-xs text-muted-foreground">{user?.email}</p>
			</div>
			<iconify-icon icon="solar:settings-linear" width="16" class="shrink-0 text-muted-foreground"></iconify-icon>
		</a>
		<button
			onclick={logout}
			class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:text-destructive hover:bg-destructive/10"
		>
			<iconify-icon icon="solar:logout-2-linear" width="16"></iconify-icon>
			Déconnexion
		</button>
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
