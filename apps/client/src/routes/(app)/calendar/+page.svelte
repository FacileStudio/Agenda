<script lang="ts">
	import { getContext } from 'svelte';
	import type { CalendarItem } from '$lib/backend';

	const app = getContext<{ calendars: CalendarItem[] }>('app');
</script>

<div class="flex h-full items-center justify-center text-muted-foreground">
	<p>Calendar view — coming soon</p>
</div>
