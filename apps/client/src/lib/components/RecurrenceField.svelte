<script lang="ts">
	import { Input, Select } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
	import { weekdayName } from '$lib/i18n/format';
	import {
		WEEKDAYS,
		buildRule,
		describeRule,
		intervalWords,
		weekdayOf,
		type Frequency,
		type Recurrence
	} from '$lib/recurrence';

	let {
		recurrence = $bindable(),
		start,
		allDay
	}: {
		recurrence: Recurrence;
		start: Date;
		allDay: boolean;
	} = $props();

	const summary = $derived(
		recurrence.freq === 'none' ? '' : describeRule(buildRule(recurrence, start, allDay), start)
	);
	const words = $derived(intervalWords(recurrence.freq));
	const editable = $derived(recurrence.freq !== 'none' && recurrence.freq !== 'custom');

	function setFrequency(value: string) {
		recurrence.freq = value as Frequency;
		if (recurrence.freq === 'WEEKLY' && recurrence.byDay.length === 0) {
			recurrence.byDay = [weekdayOf(start)];
		}
	}

	function toggleDay(day: string) {
		const has = recurrence.byDay.includes(day);
		if (has && recurrence.byDay.length === 1) return;
		recurrence.byDay = has ? recurrence.byDay.filter((d) => d !== day) : [...recurrence.byDay, day];
	}

	function setEnd(value: string) {
		recurrence.end = value as Recurrence['end'];
		if (recurrence.end === 'until' && !recurrence.until) {
			const d = new Date(start.getFullYear(), start.getMonth() + 3, start.getDate());
			recurrence.until = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
		}
	}
</script>

<div class="flex flex-col gap-3">
	<span class="text-fc-sm text-fc-fg">{t('rec.label')}</span>

	<Select value={recurrence.freq} onchange={(e) => setFrequency(e.currentTarget.value)}>
		<option value="none">{t('rec.none')}</option>
		<option value="DAILY">{t('rec.daily')}</option>
		<option value="WEEKLY">{t('rec.weekly')}</option>
		<option value="MONTHLY">{t('rec.monthly')}</option>
		<option value="YEARLY">{t('rec.yearly')}</option>
		{#if recurrence.freq === 'custom'}
			<option value="custom">{t('rec.custom')}</option>
		{/if}
	</Select>

	{#if editable}
		<div class="flex flex-col gap-3 rounded-fc-md bg-fc-component p-3">
			<div class="flex items-center gap-2 text-fc-sm text-fc-fg">
				<span class="shrink-0">{words[0]}</span>
				<Input
					type="number"
					min="1"
					max="99"
					aria-label={t('rec.interval')}
					class="h-9 w-16 px-2 text-center"
					value={String(recurrence.interval)}
					oninput={(e) => (recurrence.interval = Math.max(1, Number(e.currentTarget.value) || 1))}
				/>
				<span>{words[1]}</span>
			</div>

			{#if recurrence.freq === 'WEEKLY'}
				<div class="flex flex-wrap gap-1.5" role="group" aria-label={t('rec.weekdays')}>
					{#each WEEKDAYS as day, i (day)}
						{@const active = recurrence.byDay.includes(day)}
						<button
							type="button"
							aria-pressed={active}
							aria-label={weekdayName(i)}
							title={weekdayName(i)}
							onclick={() => toggleDay(day)}
							class="flex size-9 cursor-pointer items-center justify-center rounded-full text-fc-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring
								{active ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:text-foreground'}"
						>
							{weekdayName(i, 'narrow')}
						</button>
					{/each}
				</div>
			{/if}

			<div class="flex flex-wrap items-center gap-2 text-fc-sm text-fc-fg">
				<span class="shrink-0">{t('rec.ends')}</span>
				<Select class="h-9 w-auto" value={recurrence.end} onchange={(e) => setEnd(e.currentTarget.value)}>
					<option value="never">{t('rec.endNever')}</option>
					<option value="until">{t('rec.endUntil')}</option>
					<option value="count">{t('rec.endCount')}</option>
				</Select>
				{#if recurrence.end === 'until'}
					<Input type="date" aria-label={t('rec.endDate')} class="h-9 w-auto" bind:value={recurrence.until} />
				{:else if recurrence.end === 'count'}
					<Input
						type="number"
						min="1"
						max="999"
						aria-label={t('rec.occurrencesAria')}
						class="h-9 w-20 px-2 text-center"
						value={String(recurrence.count)}
						oninput={(e) => (recurrence.count = Math.max(1, Number(e.currentTarget.value) || 1))}
					/>
					<span>{t('rec.times')}</span>
				{/if}
			</div>
		</div>
	{/if}

	{#if summary}
		<p class="flex items-center gap-1.5 text-fc-sm text-fc-fg-muted">
			<iconify-icon icon="solar:repeat-linear" width="16" height="16" class="block size-4 shrink-0"></iconify-icon>
			{summary}
		</p>
	{/if}
</div>
