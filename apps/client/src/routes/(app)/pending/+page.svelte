<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Alert, Button, EmptyState, Skeleton, icons, toast } from '@facile/muse';
	import { ApiError, backend, notOwnSession as ownSessionRefused, type PendingAction } from '$lib/backend';
	import PendingCard from '$lib/components/PendingCard.svelte';
	import { t } from '$lib/i18n/index.svelte';

	let actions = $state<PendingAction[]>([]);
	let loading = $state(true);
	let busy = $state('');
	let notOwnSession = $state(false);

	const focused = $derived(page.url.searchParams.get('action') ?? '');
	const ordered = $derived([
		...actions.filter((action) => action.id === focused),
		...actions.filter((action) => action.id !== focused)
	]);

	onMount(load);

	/** Answers true, and shows the sign-in-again notice, when the API refused this session. */
	function refusedSession(err: unknown): boolean {
		notOwnSession = ownSessionRefused(err);
		return notOwnSession;
	}

	/** The server's own words for any other refusal, else the fallback text. */
	function reason(err: unknown, fallback: string): string {
		return err instanceof ApiError && err.status === 403 ? err.message : fallback;
	}

	/** Loads the open actions. */
	async function load() {
		loading = true;
		try {
			actions = (await backend.listPending()).actions;
		} catch (err) {
			if (!refusedSession(err)) toast.danger(reason(err, t('pending.loadFailed')));
		}
		loading = false;
	}

	/** Signs out and back in, so this browser gets its own session, then returns here. */
	async function signInAgain() {
		try {
			sessionStorage.setItem('agenda.return', page.url.pathname + page.url.search);
		} catch {}
		try {
			await backend.logout();
		} catch {}
		await goto('/login');
	}

	/** Runs one action once, then drops it from the list. */
	async function confirm(action: PendingAction) {
		busy = action.id;
		try {
			await backend.confirmPending(action.id);
			toast.success(t('pending.deleted'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			if (!refusedSession(err)) {
				toast.danger(err instanceof Error ? err.message : t('pending.failed'));
				await load();
			}
		}
		busy = '';
	}

	/** Closes one action without running it. */
	async function dismiss(action: PendingAction) {
		busy = action.id;
		try {
			await backend.dismissPending(action.id);
			toast.success(t('pending.dismissed'));
			actions = actions.filter((item) => item.id !== action.id);
		} catch (err) {
			if (!refusedSession(err)) toast.danger(err instanceof Error ? err.message : t('pending.failed'));
		}
		busy = '';
	}
</script>

<svelte:head>
	<title>{t('pending.title')}</title>
</svelte:head>

<div class="w-full px-4 py-6 md:px-8">
	<div class="mx-auto flex max-w-2xl flex-col gap-8">
		<div class="flex flex-col gap-1">
			<h1 class="text-fc-2xl font-semibold text-fc-fg">{t('pending.title')}</h1>
			<p class="text-fc-sm text-fc-fg-muted">{t('pending.intro')}</p>
		</div>

		{#if notOwnSession}
			<Alert tone="warning" title={t('pending.signInAgainTitle')}>
				<div class="flex flex-col items-start gap-3">
					<p>{t('pending.signInAgainBody')}</p>
					<Button size="sm" onclick={signInAgain}>{t('pending.signInAgain')}</Button>
				</div>
			</Alert>
		{:else if loading}
			<Skeleton class="h-48 w-full" />
		{:else if actions.length === 0}
			<EmptyState icon={icons.check} title={t('pending.none')} description={t('pending.noneBody')} />
		{:else}
			<section class="flex flex-col gap-4">
				{#each ordered as action (action.id)}
					<PendingCard {action} busy={busy === action.id} onconfirm={confirm} ondismiss={dismiss} />
				{/each}
			</section>
		{/if}
	</div>
</div>
