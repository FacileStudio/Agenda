<script lang="ts">
	import { Alert, Badge, Button, Modal } from '@facile/muse';
	import type { AgendaEvent, CalendarItem } from '$lib/backend';
	import { formatDate } from '$lib/i18n/format';
	import { t } from '$lib/i18n/index.svelte';

	let {
		open = $bindable(false),
		event,
		calendars,
		onClose
	}: { open: boolean; event: AgendaEvent | null; calendars: CalendarItem[]; onClose: () => void } = $props();

	const calendar = $derived(calendars.find((c) => c.id === event?.calendar_id));
	const due = $derived(
		event ? formatDate(event.start_at, { weekday: 'long', day: 'numeric', month: 'long', timeZone: 'UTC' }) : ''
	);
</script>

<Modal bind:open size="md" showClose {onClose} title={t('event.opusTitle')}>
	{#if event}
		<div class="flex flex-col gap-4">
			<div class="flex min-w-0 items-center gap-2">
				<h3 class="min-w-0 truncate text-fc-lg font-semibold text-fc-fg">{event.title}</h3>
				<Badge>Opus</Badge>
			</div>
			<p class="text-fc-sm text-fc-fg-muted">{t('event.due', { date: due })}</p>
			{#if calendar}
				<p class="flex items-center gap-2 text-fc-sm text-fc-fg">
					<span class="size-2.5 shrink-0 rounded-full" style="background-color: {calendar.color}"></span>
					{calendar.name}
				</p>
			{/if}
			<Alert tone="info">{t('event.opusNote')}</Alert>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
			<Button variant="secondary" class="w-full sm:w-auto" onclick={() => (open = false)}>
				{t('common.close')}
			</Button>
			{#if event?.external_url}
				<Button href={event.external_url} target="_blank" icon="solar:square-arrow-right-up-linear" class="w-full sm:w-auto">
					{t('event.openInOpus')}
				</Button>
			{/if}
		</div>
	{/snippet}
</Modal>
