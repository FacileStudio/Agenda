<script lang="ts">
	import { Tile } from '@facile/mise-en-scene';
	import OpusScene from './OpusScene.svelte';
	import PhoneScene from './PhoneScene.svelte';
	import TeamScene from './TeamScene.svelte';
	import { t } from '$lib/i18n/index.svelte';
</script>

<section class="mx-auto grid w-full max-w-5xl gap-4 px-4 py-16 sm:px-6 md:grid-cols-3">
	<Tile level={2} title={t('landing.card.opus.title')} body={t('landing.card.opus.text')}><OpusScene /></Tile>
	<Tile level={2} title={t('landing.card.team.title')} body={t('landing.card.team.text')}><TeamScene /></Tile>
	<Tile level={2} title={t('landing.card.phone.title')} body={t('landing.card.phone.text')}><PhoneScene /></Tile>
</section>
