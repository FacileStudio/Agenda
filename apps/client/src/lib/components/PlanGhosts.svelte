<script lang="ts">
	import { Tooltip } from '@facile/muse';
	import type { CalendarItem, PlanBlock } from '$lib/backend';
	import { calendarColor } from '$lib/calendar-colors';
	import { formatTime } from '$lib/i18n/format';
	import { t } from '$lib/i18n/index.svelte';

	let {
		blocks,
		dropped = [],
		selected = null,
		calendars,
		slotHeight,
		onSelect
	}: {
		blocks: PlanBlock[];
		dropped?: number[];
		selected?: number | null;
		calendars: CalendarItem[];
		slotHeight: number;
		onSelect: (block: PlanBlock) => void;
	} = $props();

	function minutesOf(date: Date): number {
		return date.getHours() * 60 + date.getMinutes();
	}

	function place(block: PlanBlock): string {
		const start = new Date(block.start_at);
		const end = new Date(block.end_at);
		const from = minutesOf(start);
		const to = end.getDate() === start.getDate() ? minutesOf(end) : 24 * 60;
		const height = (Math.max(to - from, 30) / 60) * slotHeight;
		return `top: ${(from / 60) * slotHeight + 1}px; height: ${Math.max(height - 2, 0)}px;`;
	}

	function fill(block: PlanBlock): string {
		const level = dropped.includes(block.id) ? 12 : selected === block.id ? 62 : 38;
		return `background-color: color-mix(in srgb, ${calendarColor(calendars, block.calendar_id)} ${level}%, transparent);`;
	}

	function label(block: PlanBlock): string {
		return t('bulle.ghost.aria', { title: block.title, time: formatTime(block.start_at) });
	}
</script>

{#each blocks as block (block.id)}
	<Tooltip text={block.reason || block.title} describe={false} class="contents">
		<button
			type="button"
			aria-label={label(block)}
			onpointerdown={(e) => e.stopPropagation()}
			onclick={() => onSelect(block)}
			class="absolute right-2 left-0.5 z-20 flex cursor-pointer flex-col items-start overflow-hidden rounded-md px-1.5 py-1 text-left text-xs leading-tight font-medium text-fc-fg transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring {dropped.includes(block.id) ? 'line-through' : ''}"
			style="{place(block)} {fill(block)}"
		>
			<span class="w-full min-w-0 truncate font-semibold">{block.title}</span>
			<span class="w-full truncate opacity-80">{formatTime(block.start_at)}</span>
		</button>
	</Tooltip>
{/each}
