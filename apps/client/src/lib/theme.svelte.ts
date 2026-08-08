import { browser } from '$app/environment';

export type ThemePreference = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'agenda.theme';

function isPreference(value: string | null): value is ThemePreference {
	return value === 'light' || value === 'dark' || value === 'system';
}

class ThemeState {
	preference = $state<ThemePreference>('system');
	resolved = $state<'light' | 'dark'>('light');

	/** Reads the stored preference and starts tracking the OS setting. */
	restore() {
		if (!browser) return;

		const stored = localStorage.getItem(STORAGE_KEY);
		this.preference = isPreference(stored) ? stored : 'system';
		this.apply();

		const query = window.matchMedia('(prefers-color-scheme: dark)');
		query.addEventListener('change', () => {
			if (this.preference === 'system') this.apply();
		});
	}

	set(next: ThemePreference) {
		this.preference = next;
		if (!browser) return;
		localStorage.setItem(STORAGE_KEY, next);
		this.apply();
	}

	/**
	 * Sets both classes, never just `dark`. muse paints dark from
	 * `@media (prefers-color-scheme: dark)` scoped to `:root:not(.light)`, so on a
	 * dark-preferring OS removing `dark` is not enough to get light back — only an
	 * explicit `light` class escapes the media block. Agenda additionally has 41
	 * `dark:` Tailwind utilities that only fire off an explicit `.dark` class, so an
	 * explicit class must always be present — never leave both off.
	 */
	private apply() {
		if (!browser) return;
		const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
		const dark = this.preference === 'dark' || (this.preference === 'system' && prefersDark);
		this.resolved = dark ? 'dark' : 'light';
		document.documentElement.classList.toggle('dark', dark);
		document.documentElement.classList.toggle('light', !dark);
	}
}

export const theme = new ThemeState();
