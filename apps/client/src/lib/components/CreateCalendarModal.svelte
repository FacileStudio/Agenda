<script lang="ts">
	import { Alert, Button, SwatchPicker, Field, Input, Modal, icons } from '@facile/muse';
	import { backend } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let {
		open = $bindable(false),
		onCreated,
		onClose
	}: {
		open: boolean;
		onCreated: () => void;
		onClose: () => void;
	} = $props();

	const COLORS = [
		'#ef4444', '#f97316', '#eab308', '#22c55e',
		'#06b6d4', '#3b82f6', '#8b5cf6', '#ec4899'
	];

	let name = $state('');
	let color = $state('#3b82f6');
	let description = $state('');
	let echoUrl = $state('');
	let saving = $state(false);
	let error = $state('');

	$effect(() => {
		if (open) {
			name = '';
			color = '#3b82f6';
			description = '';
			echoUrl = '';
			error = '';
		}
	});

	async function handleCreate() {
		if (!name.trim()) { error = t('calendarModal.nameRequired'); return; }
		saving = true; error = '';
		try {
			await backend.createCalendar({ name: name.trim(), color, description: description || undefined, echo_url: echoUrl.trim() || undefined });
			onCreated();
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : t('common.genericError');
		} finally {
			saving = false;
		}
	}
</script>

<Modal bind:open title={t('calendarModal.newTitle')} showClose {onClose}>
	<div class="flex flex-col gap-4">
		<Field label={t('common.name')}>
			<Input bind:value={name} placeholder={t('calendarModal.namePlaceholder')} autofocus />
		</Field>

		<div class="flex flex-col gap-1.5">
			<span class="text-fc-sm text-fc-fg">{t('common.color')}</span>
			<SwatchPicker bind:value={color} colors={COLORS} label={t('calendarModal.colorLabel')} />
		</div>

		<Field label={t('common.description')}>
			<Input bind:value={description} placeholder={t('calendarModal.descriptionPlaceholder')} />
		</Field>

		<Field label={t('calendarModal.echo')} helper={t('calendarModal.echoHelper')}>
			<Input bind:value={echoUrl} placeholder="https://echo.facile.studio" />
		</Field>

		{#if error}
			<Alert tone="danger">{error}</Alert>
		{/if}
	</div>

	{#snippet footer()}
		<div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
			<Button variant="outline" class="w-full sm:w-auto" onclick={() => (open = false)} disabled={saving}>
				{t('common.cancel')}
			</Button>
			<Button
				class="w-full sm:w-auto"
				icon={icons.plus}
				onclick={handleCreate}
				disabled={saving}
			>
				{saving ? t('common.creating') : t('common.create')}
			</Button>
		</div>
	{/snippet}
</Modal>
