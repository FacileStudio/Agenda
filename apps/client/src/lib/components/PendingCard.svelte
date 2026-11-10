<script lang="ts">
	import { Badge, Button, Card, icons } from '@facile/muse';
	import type { PendingAction } from '$lib/backend';
	import { formatDate } from '$lib/i18n/format';
	import { t } from '$lib/i18n/index.svelte';

	let {
		action,
		busy,
		onconfirm,
		ondismiss
	}: {
		action: PendingAction;
		busy: boolean;
		onconfirm: (action: PendingAction) => void;
		ondismiss: (action: PendingAction) => void;
	} = $props();

	const event = $derived(action.event);
	const when = $derived.by(() => {
		if (!event) return '';
		if (event.is_all_day) return formatDate(event.start_at, { dateStyle: 'full' });
		return `${formatDate(event.start_at, { dateStyle: 'full', timeStyle: 'short' })} – ${formatDate(event.end_at, { timeStyle: 'short' })}`;
	});

	/** Names the client that asked: the host of a client URL, else its id. */
	function clientName(clientId: string): string {
		if (clientId === 'bulle') return 'Bulle';
		try {
			return new URL(clientId).host;
		} catch {
			return clientId || '?';
		}
	}
</script>

<Card class="flex flex-col gap-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<Badge tone="danger">{t('pending.deleteEvent')}</Badge>
		<span class="text-fc-xs text-fc-fg-muted">
			{t('pending.askedBy', { client: clientName(action.client_id) })} · {formatDate(action.created_at, {
				day: 'numeric',
				month: 'short',
				hour: '2-digit',
				minute: '2-digit'
			})}
		</span>
	</div>

	{#if event}
		<div class="flex flex-col gap-1 rounded-fc-md bg-fc-surface p-4">
			<p class="break-words text-fc-md font-medium text-fc-fg">{event.title}</p>
			<p class="text-fc-sm text-fc-fg-muted">{when}</p>
			{#if event.location}
				<p class="break-words text-fc-sm text-fc-fg-muted">{event.location}</p>
			{/if}
			{#if event.recurring}
				<p class="text-fc-sm text-fc-warning">{t('pending.recurring')}</p>
			{/if}
		</div>
	{:else}
		<p class="break-words text-fc-sm text-fc-fg-muted">{action.summary}</p>
	{/if}

	{#if action.state === 'changed'}
		<p class="text-fc-sm text-fc-warning">{t('pending.changed')}</p>
	{:else if action.state === 'missing'}
		<p class="text-fc-sm text-fc-warning">{t('pending.missing')}</p>
	{/if}

	<div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
		<Button variant="ghost" class="w-full sm:w-auto" disabled={busy} onclick={() => ondismiss(action)}>
			{t('pending.dismiss')}
		</Button>
		{#if action.state === 'ready'}
			<Button
				variant="danger"
				class="w-full sm:w-auto"
				disabled={busy}
				icon={icons.remove}
				onclick={() => onconfirm(action)}
			>
				{t('pending.confirmDelete')}
			</Button>
		{/if}
	</div>
</Card>
