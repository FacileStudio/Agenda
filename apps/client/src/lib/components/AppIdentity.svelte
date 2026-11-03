<script lang="ts">
	import { getContext } from 'svelte';
	import { SpaceSwitcher } from '@facile/muse';
	import UserMenu from '$lib/components/UserMenu.svelte';

	type Identity = {
		navUser: { name: string; email: string; avatar?: string };
		switcherSpaces: { id: string; name: string }[];
		activeSpaceId: string | null;
		selectSpace: (id: string | null) => void;
	};

	const app = getContext<Identity>('app');
</script>

<div class="flex min-w-0 shrink-0 items-center gap-2">
	<UserMenu name={app.navUser.name} email={app.navUser.email} avatar={app.navUser.avatar} />
	{#if app.switcherSpaces.length > 0}
		<SpaceSwitcher
			class="w-44 sm:w-56"
			spaces={app.switcherSpaces}
			activeId={app.activeSpaceId}
			onSelect={app.selectSpace}
			personalLabel="Personnel"
			manageHref="/spaces"
			manageLabel="Gérer les espaces"
		/>
	{/if}
</div>
