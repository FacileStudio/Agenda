<script lang="ts">
	import { page } from '$app/stores';
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Avatar, Button, IconButton } from '@facile/muse';
	import { backend, registreSpaceUrl, resolveFileUrl, type SpaceItem, type SpaceMember } from '$lib/backend';
	import { setCalendarFilter } from '$lib/calendar-filter.svelte';
	import type { CalendarItem } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';
	import { formatDate } from '$lib/i18n/format';

	const spaceId = $derived(Number($page.params.id));

	let space = $state<SpaceItem | null>(null);
	let members = $state<SpaceMember[]>([]);
	let loading = $state(true);

	const canManage = $derived(space?.role === 'owner' || space?.role === 'admin');
	const app = getContext<{ calendars: CalendarItem[] }>('app');
	const spaceCalendar = $derived(app.calendars.find((c) => c.space_id === spaceId));

	function roleBadgeClass(role: string): string {
		switch (role) {
			case 'owner':
				return 'bg-fc-owner/10 text-fc-owner';
			case 'admin':
				return 'bg-fc-admin/10 text-fc-admin';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}

	const roleLabelKeys = { owner: 'role.owner', admin: 'role.admin', member: 'role.member' } as const;

	function roleLabel(role: string): string {
		return role in roleLabelKeys ? t(roleLabelKeys[role as keyof typeof roleLabelKeys]) : role;
	}

	onMount(async () => {
		try {
			const [s, m] = await Promise.all([
				backend.getSpace(spaceId),
				backend.listSpaceMembers(spaceId)
			]);
			space = s;
			members = m;
		} catch {
			goto('/spaces');
		} finally {
			loading = false;
		}
	});

	function openCalendar() {
		if (!spaceCalendar) return;
		setCalendarFilter(spaceCalendar.id);
		goto('/calendar');
	}
</script>

<svelte:head>
	<title>{space?.name ?? t('spaces.fallbackTitle')} — Agenda</title>
</svelte:head>

<div class="flex h-full flex-col">
	{#if loading}
		<div class="flex flex-1 items-center justify-center">
			<iconify-icon icon="solar:refresh-linear" width="20" class="animate-spin text-muted-foreground"></iconify-icon>
		</div>
	{:else if space}
		<div class="px-4 py-4 md:px-8 md:py-5">
			<div class="flex flex-wrap items-center gap-3">
				<IconButton variant="ghost" onclick={() => goto('/spaces')} aria-label={t('spaces.back')}>
					<iconify-icon icon="solar:arrow-left-linear" width="20"></iconify-icon>
				</IconButton>
				<div class="min-w-0 flex-1">
					<h1 class="truncate text-lg font-semibold">{space.name}</h1>
					{#if space.description}
						<p class="mt-1 truncate text-sm text-muted-foreground">{space.description}</p>
					{/if}
				</div>
				<div class="flex basis-full flex-wrap items-center gap-2 sm:basis-auto sm:shrink-0">
					{#if spaceCalendar}
						<Button onclick={openCalendar} icon="solar:calendar-linear">
							{t('spaces.viewCalendar')}
						</Button>
					{/if}
					<Button
						href={registreSpaceUrl(space.registre_space_id)}
						target="_blank"
						rel="noopener"
						variant="secondary"
						icon="solar:square-top-down-linear"
					>
						{canManage ? t('common.manageInRegistre') : t('spaces.viewInRegistre')}
					</Button>
				</div>
			</div>
		</div>

		<div class="flex-1 overflow-auto p-4 md:p-8">
			<div class="max-w-2xl space-y-6">
				<div class="rounded-lg bg-muted/40 p-4">
					<p class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">{t('spaces.info')}</p>
					<div class="mt-3 grid gap-2 text-sm">
						<div class="flex justify-between">
							<span class="text-muted-foreground">{t('spaces.role')}</span>
							<span class="rounded-full px-2.5 py-0.5 text-xs {roleBadgeClass(space.role)}">
								{roleLabel(space.role)}
							</span>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">{t('common.members')}</span>
							<span>{members.length}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">{t('spaces.createdOn')}</span>
							<span>{formatDate(space.created_at)}</span>
						</div>
					</div>
				</div>

				<div>
					<div class="flex items-center justify-between">
						<p class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">{t('common.members')}</p>
						{#if canManage}
							<Button
								href={registreSpaceUrl(space.registre_space_id)}
								target="_blank"
								rel="noopener"
								variant="secondary"
								size="sm"
								icon="solar:users-group-rounded-linear"
							>
								{t('spaces.inviteInRegistre')}
							</Button>
						{/if}
					</div>

					<div class="mt-3 grid gap-2">
						{#each members as member (member.user_id)}
							<div class="flex items-center gap-3 rounded-lg bg-muted/40 p-3">
								<Avatar
									class="size-9"
									src={member.avatar_url ? resolveFileUrl(member.avatar_url) : undefined}
									name={member.name || member.email}
								/>
								<div class="min-w-0 flex-1">
									<p class="truncate text-sm font-medium">{member.name || member.email}</p>
									{#if member.name}
										<p class="truncate text-xs text-muted-foreground">{member.email}</p>
									{/if}
								</div>
								<span class="shrink-0 rounded-full px-2.5 py-0.5 text-xs {roleBadgeClass(member.role)}">
									{roleLabel(member.role)}
								</span>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
