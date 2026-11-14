<script lang="ts">
	import { ForDevelopers, icons, type DeveloperRow } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';

	const rows: DeveloperRow[] = $derived([
		{ icon: icons.refresh, title: 'CalDAV', text: t('landing.dev.caldav') },
		{ kind: 'cli', text: t('landing.dev.cli'), command: 'facile install agenda' },
		{ kind: 'api', text: t('landing.dev.api'), href: 'https://agenda.facile.studio/docs' },
		{ kind: 'mcp', text: t('landing.dev.mcp') },
		{ kind: 'source', text: t('landing.dev.source'), href: 'https://github.com/FacileStudio/Agenda' }
	]);
</script>

<div id="developers" class="mx-auto w-full max-w-5xl scroll-mt-20 px-4 py-16 sm:px-6">
	<ForDevelopers {rows} description={t('landing.dev.line')} selfHostHref={t('landing.dev.selfHost')} />
</div>
