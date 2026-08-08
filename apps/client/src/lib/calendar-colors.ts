import type { CalendarItem } from '$lib/backend';

const FALLBACK = '#6B7280';

/**
 * Ink candidates, matching muse's `--fc-fg` in each mode. They are used as literal
 * values rather than tokens on purpose: a calendar colour is caller-supplied data, and
 * every `fc-*` foreground inverts with the theme, so `text-foreground` on an event chip
 * paints white ink on a pale calendar for any dark-mode user.
 */
const LIGHT_INK = 'oklch(0.985 0 0)';
const DARK_INK = 'oklch(0.145 0 0)';
const LIGHT_INK_LUMA = 0.955;
const DARK_INK_LUMA = 0.0145;

/**
 * Resolves a calendar's stored colour, falling back to a neutral grey when the calendar
 * is missing or its colour column is empty.
 */
export function calendarColor(calendars: CalendarItem[], calendarId: number): string {
	return calendars.find((c) => c.id === calendarId)?.color || FALLBACK;
}

function toLinear(channel: number): number {
	return channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4;
}

/** WCAG relative luminance of a `#rgb` or `#rrggbb` colour; `0` for anything unparseable. */
function luminance(hex: string): number {
	const raw = hex.trim().replace(/^#/, '');
	const full = raw.length === 3 ? raw.replace(/./g, (c) => c + c) : raw;
	if (!/^[0-9a-fA-F]{6}$/.test(full)) return 0;
	const [r, g, b] = [0, 2, 4].map((i) => toLinear(parseInt(full.slice(i, i + 2), 16) / 255));
	return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

/**
 * Picks the ink that contrasts best with a caller-supplied fill. Returns a CSS colour to
 * be set inline — never a token class, which would invert with the theme while the fill
 * underneath it stays put.
 */
export function inkOn(fill: string): string {
	const luma = luminance(fill);
	const withLight = (LIGHT_INK_LUMA + 0.05) / (luma + 0.05);
	const withDark = (luma + 0.05) / (DARK_INK_LUMA + 0.05);
	return withLight >= withDark ? LIGHT_INK : DARK_INK;
}
