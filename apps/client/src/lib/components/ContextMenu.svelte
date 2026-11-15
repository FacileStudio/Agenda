<script lang="ts">
	import { Icon, type MenuAction } from '@facile/muse';
	import { contextMenu } from '$lib/context-menu.svelte';

	let panel = $state<HTMLElement | null>(null);
	let left = $state(0);
	let top = $state(0);

	const actions = $derived(
		contextMenu.items.filter((entry): entry is MenuAction => !('separator' in entry))
	);
	const iconColumn = $derived(actions.some((a) => a.icon));

	$effect(() => {
		if (!contextMenu.open || !panel) return;
		const margin = 8;
		const { width, height } = panel.getBoundingClientRect();
		left = Math.max(margin, Math.min(contextMenu.x, window.innerWidth - width - margin));
		top = Math.max(margin, Math.min(contextMenu.y, window.innerHeight - height - margin));
		panel.querySelector<HTMLElement>('[role="menuitem"]:not([disabled])')?.focus();
	});

	$effect(() => {
		if (!contextMenu.open) return;
		const outside = (e: Event) => {
			if (!panel?.contains(e.target as Node)) contextMenu.hide();
		};
		window.addEventListener('pointerdown', outside, true);
		window.addEventListener('scroll', contextMenu.hide, true);
		window.addEventListener('resize', contextMenu.hide);
		window.addEventListener('blur', contextMenu.hide);
		return () => {
			window.removeEventListener('pointerdown', outside, true);
			window.removeEventListener('scroll', contextMenu.hide, true);
			window.removeEventListener('resize', contextMenu.hide);
			window.removeEventListener('blur', contextMenu.hide);
		};
	});

	function choose(action: MenuAction) {
		if (action.disabled) return;
		contextMenu.hide();
		action.onselect?.();
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			contextMenu.hide();
			return;
		}
		if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp' && e.key !== 'Home' && e.key !== 'End') return;
		e.preventDefault();
		const rows = Array.from(panel?.querySelectorAll<HTMLElement>('[role="menuitem"]:not([disabled])') ?? []);
		if (rows.length === 0) return;
		const current = rows.indexOf(document.activeElement as HTMLElement);
		const next =
			e.key === 'Home'
				? 0
				: e.key === 'End'
					? rows.length - 1
					: e.key === 'ArrowDown'
						? (current + 1) % rows.length
						: (current - 1 + rows.length) % rows.length;
		rows[next].focus();
	}
</script>

{#if contextMenu.open}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		bind:this={panel}
		role="menu"
		tabindex="-1"
		class="fixed z-[100] min-w-48 rounded-fc-md bg-fc-bg p-1 text-fc-fg shadow-lg"
		style="left: {left}px; top: {top}px;"
		oncontextmenu={(e) => e.preventDefault()}
		onkeydown={onKeydown}
	>
		{#each contextMenu.items as entry, i (i)}
			{#if 'separator' in entry}
				<div role="separator" class="h-1.5"></div>
			{:else}
				<button
					type="button"
					role="menuitem"
					disabled={entry.disabled}
					class="flex min-h-9 w-full cursor-pointer items-center gap-2.5 rounded-fc-sm px-2.5 py-1.5 text-left text-fc-md transition-colors hover:bg-fc-surface focus-visible:bg-fc-surface focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 {entry.tone === 'danger' ? 'text-fc-danger' : 'text-fc-fg'}"
					onclick={() => choose(entry)}
				>
					{#if entry.icon}<Icon icon={entry.icon} size={16} />{:else if iconColumn}<span class="size-4 shrink-0" aria-hidden="true"></span>{/if}
					<span class="min-w-0 flex-1 truncate">{entry.label}</span>
				</button>
			{/if}
		{/each}
	</div>
{/if}
