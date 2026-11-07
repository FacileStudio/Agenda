<script lang="ts">
	import { onMount } from 'svelte';
	import { TokensSection, type TokenItem, toast } from '@facile/muse';
	import { backend, type ApiTokenStatus } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let apiToken = $state<ApiTokenStatus | null>(null);

	const tokens = $derived<TokenItem[]>(
		apiToken?.has_token ? [{ id: 'caldav', name: apiToken.name ?? 'Token' }] : []
	);

	onMount(async () => {
		try {
			apiToken = await backend.getApiToken();
		} catch {
			apiToken = null;
		}
	});

	async function createToken(draft: { name: string }) {
		const wasReplacing = Boolean(apiToken?.has_token);
		const created = await backend.createApiToken(draft.name || 'CalDAV');
		apiToken = { has_token: true, name: created.name, created_at: created.created_at };
		toast.success(wasReplacing ? t('api.toastReplaced') : t('api.toastGenerated'));
		return created.token;
	}

	async function revokeToken() {
		try {
			await backend.deleteApiToken();
			apiToken = { has_token: false };
			toast.success(t('api.toastRevoked'));
		} catch (error: unknown) {
			toast.danger(error instanceof Error ? error.message : t('common.error'));
			throw error;
		}
	}
</script>

<TokensSection {tokens} onCreate={createToken} onRevoke={revokeToken} description={t('api.tokenText')} />
