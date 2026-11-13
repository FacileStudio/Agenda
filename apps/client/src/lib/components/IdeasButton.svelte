<script lang="ts">
	import { Badge, Button, Tooltip, getI18n, ideasNavItem } from '@facile/muse';
	import { page } from '$app/state';
	import { ideasStore } from '$lib/ideas.svelte';

	let { class: className = '' }: { class?: string } = $props();

	const i18n = getI18n();
	const entry = $derived(
		ideasNavItem({
			href: '/ideas',
			active: page.url.pathname.startsWith('/ideas'),
			count: ideasStore.count,
			running: ideasStore.running,
			i18n
		})
	);
</script>

{#if ideasStore.available}
	<Tooltip text={entry.label} describe={false} class="contents">
		<Button
			href={entry.href}
			variant={entry.active ? 'secondary' : 'ghost'}
			size="lg"
			icon={entry.icon}
			class={className}
			aria-label={entry.hint ? `${entry.label}, ${entry.hint}` : entry.label}
			aria-current={entry.active ? 'page' : undefined}
		>
			<span class="hidden sm:inline">{entry.label}</span>
			{#if entry.badge}
				<Badge tone="accent">{entry.badge}</Badge>
			{/if}
			{#if entry.busy}
				<span class="size-2 shrink-0 animate-pulse rounded-full bg-fc-accent motion-reduce:animate-none"></span>
			{/if}
		</Button>
	</Tooltip>
{/if}
