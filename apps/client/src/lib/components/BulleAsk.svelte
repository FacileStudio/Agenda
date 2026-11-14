<script lang="ts">
	import { Icon, icons } from '@facile/muse';
	import { Stream } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';
	import { LANDING_ICONS } from '$lib/landing-icons';

	let {
		draft,
		sent,
		words,
		style
	}: {
		draft: string;
		sent: boolean;
		words: number;
		style: string;
	} = $props();
</script>

<div class="ba" {style}>
	<div class="ba-head">
		<span class="ba-mark"><Icon icon={LANDING_ICONS.bulle} size={16} /></span>
		Bulle
		<span class="ba-pro">Pro</span>
	</div>
	<div class="ba-thread">
		<p class="ba-ask" class:ba-shown={sent}>{t('landing.demo.prompt')}</p>
		<Stream class="ba-reply" text={t('landing.demo.reply')} count={words} />
	</div>
	<div class="ba-composer">
		<span class="ba-draft" class:ba-placeholder={!draft}>{draft || t('landing.demo.ask')}</span>
		<span class="ba-send"><Icon icon={icons.arrow} size={16} /></span>
	</div>
</div>

<style>
	.ba {
		position: absolute;
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 14px;
		border-radius: 18px;
		background: var(--color-fc-component);
		box-shadow: var(--mes-shadow-hero, 0 30px 70px -20px rgb(0 0 0 / 0.22), 0 8px 24px -8px rgb(0 0 0 / 0.1));
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
		font-size: 13px;
		line-height: 1.35;
	}
	@media (prefers-color-scheme: dark) {
		:global(:root:not(.light)) .ba {
			background: color-mix(in oklch, var(--color-fc-component) 88%, var(--color-fc-fg));
		}
	}
	:global(:root.dark) .ba {
		background: color-mix(in oklch, var(--color-fc-component) 88%, var(--color-fc-fg));
	}
	.ba-head {
		display: flex;
		align-items: center;
		gap: 8px;
		font-weight: 700;
	}
	.ba-pro {
		margin-left: auto;
		padding: 2px 8px;
		border-radius: 999px;
		background: var(--color-fc-surface);
		color: var(--color-fc-fg-muted);
		font-size: 11px;
		white-space: nowrap;
	}
	.ba-mark {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		border-radius: 999px;
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
	}
	.ba-thread {
		display: flex;
		flex: 1;
		flex-direction: column;
		gap: 8px;
		min-height: 0;
	}
	.ba-ask {
		align-self: flex-end;
		max-width: 88%;
		margin: 0;
		padding: 7px 12px;
		border-radius: 14px;
		background: var(--color-fc-surface);
		opacity: 0;
		transform: translateY(6px);
		transition:
			opacity 250ms ease,
			transform 250ms ease;
	}
	.ba-shown {
		opacity: 1;
		transform: none;
	}
	.ba :global(.ba-reply) {
		margin: 0;
		color: var(--color-fc-fg);
	}
	.ba-composer {
		display: flex;
		align-items: center;
		gap: 8px;
		height: 40px;
		padding: 0 6px 0 14px;
		border-radius: 999px;
		background: var(--color-fc-surface);
	}
	.ba-draft {
		flex: 1;
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.ba-placeholder {
		color: var(--color-fc-fg-muted);
	}
	.ba-send {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 30px;
		height: 30px;
		border-radius: 999px;
		background: var(--color-fc-accent);
		color: var(--color-fc-accent-fg);
	}
</style>
