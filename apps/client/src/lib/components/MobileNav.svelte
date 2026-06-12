<script lang="ts">
	import { page } from '$app/stores';

	const items = [
		{ href: '/calendar', label: 'Calendrier', icon: 'solar:calendar-linear' },
		{ href: '/settings', label: 'Réglages', icon: 'solar:settings-minimalistic-linear' }
	];
</script>

<nav
	class="fixed inset-x-0 z-50 flex justify-center px-4 md:hidden"
	style="bottom: max(0.75rem, env(safe-area-inset-bottom))"
>
	<div
		class="flex items-center gap-1 rounded-2xl border border-border/70 bg-background/90 p-1.5 shadow-xl backdrop-blur-md"
	>
		{#each items as item (item.href)}
			{@const active = $page.url.pathname.startsWith(item.href)}
			<a
				href={item.href}
				class="flex min-w-20 flex-col items-center gap-0.5 rounded-xl px-4 py-1.5 text-[10px] font-medium transition-colors {active
					? 'bg-foreground text-background'
					: 'text-muted-foreground hover:text-foreground'}"
			>
				<iconify-icon icon={item.icon} width="20"></iconify-icon>
				<span>{item.label}</span>
			</a>
		{/each}
	</div>
</nav>
