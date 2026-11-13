<script lang="ts">
	import { Avatar, Tooltip } from '@facile/muse';
	import { resolveFileUrl, type EventCreator } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let {
		creator,
		class: className = ''
	}: {
		creator: EventCreator | null;
		class?: string;
	} = $props();

	const name = $derived(creator ? creator.name || creator.email : '');
</script>

{#if creator}
	<Tooltip text={t('calendar.createdBy', { name })} class="contents">
		<Avatar
			size="sm"
			class="size-4 text-[0.55rem]! leading-none {className}"
			src={creator.avatar_url ? resolveFileUrl(creator.avatar_url) : undefined}
			{name}
		/>
	</Tooltip>
{/if}
