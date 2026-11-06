import { locale } from './index.svelte';

/** Formats a date or instant with `Intl.DateTimeFormat` in the current locale. */
export function formatDate(value: Date | string | number, options: Intl.DateTimeFormatOptions = {}): string {
	return new Intl.DateTimeFormat(locale(), options).format(new Date(value));
}

/** Formats the clock time of an instant in the current locale. */
export function formatTime(value: Date | string | number): string {
	return formatDate(value, { hour: '2-digit', minute: '2-digit' });
}

/** Upper-cases the first letter only, as a heading wants it in every locale. */
export function capitalize(text: string): string {
	return text.charAt(0).toLocaleUpperCase(locale()) + text.slice(1);
}

/** The name of a weekday, Monday being index 0, in the current locale. */
export function weekdayName(index: number, width: 'long' | 'short' | 'narrow' = 'long'): string {
	return formatDate(new Date(2024, 0, 1 + index), { weekday: width });
}

/** Compares two names the way the current locale sorts them. */
export function compareNames(a: string, b: string): number {
	return a.localeCompare(b, locale());
}
