import { browser } from '$app/environment';
import { createTranslator, LOCALE_COOKIE, resolveLocale, type Locale } from '@facile/muse';
import { catalogs } from './messages';

export const MONDAY_FIRST = 'fr-FR';

const session = $state<{ locale: string | null }>({ locale: null });
let active: () => Locale = () => 'en';

/** Records the locale the signed-in person has on their Registre profile. */
export function setSessionLocale(locale: string | null) {
	session.locale = locale;
}

function cookieLocale(): string | null {
	if (!browser) return null;
	const prefix = `${LOCALE_COOKIE}=`;
	const found = document.cookie.split('; ').find((part) => part.startsWith(prefix));
	return found ? decodeURIComponent(found.slice(prefix.length)) : null;
}

/** The locale source for `provideLocale`: session, then the picked cookie, then the browser. */
export function localeSource(): string | null {
	return session.locale ?? cookieLocale() ?? (browser ? navigator.language : null);
}

/** Persists the locale picked on a session-less page for the next visit. */
export function rememberLocale(locale: Locale) {
	document.cookie = `${LOCALE_COOKIE}=${locale}; path=/; max-age=31536000; samesite=lax`;
}

/** Points the translator and the Intl helpers at the locale `provideLocale` resolved. */
export function bindLocale(source: () => Locale) {
	active = source;
}

/** The locale currently in force. */
export function locale(): Locale {
	return resolveLocale(active());
}

/** Translates an app message in the current locale. */
export const t = createTranslator(catalogs, () => active());
