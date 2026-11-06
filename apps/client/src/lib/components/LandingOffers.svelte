<script lang="ts">
	import { PricingPlans } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
</script>

<section class="mx-auto max-w-5xl px-6 py-16 empty:hidden sm:py-20">
	<PricingPlans highlight={[]} title={t('landing.pricing')} />
</section>
