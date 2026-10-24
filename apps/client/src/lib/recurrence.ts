import type { AgendaEvent } from '$lib/backend';

export type Frequency = 'none' | 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'YEARLY' | 'custom';
export type RecurrenceEnd = 'never' | 'until' | 'count';

export type Recurrence = {
	freq: Frequency;
	interval: number;
	byDay: string[];
	end: RecurrenceEnd;
	until: string;
	count: number;
	raw: string;
};

export const WEEKDAYS = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU'];
export const WEEKDAY_LETTERS = ['L', 'M', 'M', 'J', 'V', 'S', 'D'];
const WEEKDAY_NAMES = ['lundi', 'mardi', 'mercredi', 'jeudi', 'vendredi', 'samedi', 'dimanche'];
const MODELLED_PARTS = new Set(['FREQ', 'INTERVAL', 'BYDAY', 'UNTIL', 'COUNT', 'WKST']);

function pad(n: number): string {
	return String(n).padStart(2, '0');
}

/** Returns the RRULE weekday code (MO..SU) of a local date. */
export function weekdayOf(date: Date): string {
	return WEEKDAYS[(date.getDay() + 6) % 7];
}

/** Keys an event occurrence: a recurring series shares one ID across its occurrences. */
export function eventKey(event: AgendaEvent): string {
	return `${event.id}-${event.start_at}`;
}

function untilToLocalDate(value: string): string {
	if (/^\d{8}$/.test(value)) return `${value.slice(0, 4)}-${value.slice(4, 6)}-${value.slice(6, 8)}`;
	const m = value.match(/^(\d{4})(\d{2})(\d{2})T(\d{2})(\d{2})(\d{2})(Z?)$/);
	if (!m) return '';
	const d = m[7]
		? new Date(Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], +m[6]))
		: new Date(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], +m[6]);
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function parts(rule: string): Map<string, string> {
	const map = new Map<string, string>();
	for (const part of rule.replace(/^RRULE:/i, '').split(';')) {
		const [k, v] = part.split('=');
		if (k && v !== undefined) map.set(k.toUpperCase(), v.toUpperCase());
	}
	return map;
}

/**
 * Reads an RRULE into the form state. A rule using parts the form cannot edit
 * (BYMONTHDAY, BYSETPOS, HOURLY...) comes back as `custom` with `raw` kept, so
 * saving the event leaves a CalDAV client's rule untouched.
 */
export function parseRule(rule: string, start: Date): Recurrence {
	const base: Recurrence = {
		freq: 'none',
		interval: 1,
		byDay: [weekdayOf(start)],
		end: 'never',
		until: '',
		count: 10,
		raw: rule
	};
	if (!rule) return base;
	const p = parts(rule);
	const freq = p.get('FREQ') ?? '';
	const modelled = ['DAILY', 'WEEKLY', 'MONTHLY', 'YEARLY'].includes(freq);
	const byDay = p.get('BYDAY')?.split(',') ?? [];
	const plainDays = byDay.every((d) => WEEKDAYS.includes(d));
	if (!modelled || [...p.keys()].some((k) => !MODELLED_PARTS.has(k)) || !plainDays || (byDay.length > 0 && freq !== 'WEEKLY')) {
		return { ...base, freq: 'custom' };
	}
	const until = p.get('UNTIL');
	const count = p.get('COUNT');
	return {
		...base,
		freq: freq as Frequency,
		interval: Math.max(1, Number(p.get('INTERVAL') ?? 1) || 1),
		byDay: byDay.length ? byDay : base.byDay,
		end: until ? 'until' : count ? 'count' : 'never',
		until: until ? untilToLocalDate(until) : '',
		count: count ? Math.max(1, Number(count) || 1) : base.count
	};
}

/**
 * Writes the form state back to an RRULE value. UNTIL is the end of the chosen
 * local day, as a DATE for all-day events and a UTC DATE-TIME otherwise, which
 * is what RFC 5545 requires next to a zoned DTSTART.
 */
export function buildRule(r: Recurrence, start: Date, allDay: boolean): string {
	if (r.freq === 'none') return '';
	if (r.freq === 'custom') return r.raw;
	const out = [`FREQ=${r.freq}`];
	if (r.interval > 1) out.push(`INTERVAL=${Math.floor(r.interval)}`);
	if (r.freq === 'WEEKLY') {
		const days = r.byDay.length ? r.byDay : [weekdayOf(start)];
		out.push(`BYDAY=${WEEKDAYS.filter((d) => days.includes(d)).join(',')}`);
	}
	if (r.end === 'count' && r.count >= 1) out.push(`COUNT=${Math.floor(r.count)}`);
	if (r.end === 'until' && r.until) {
		if (allDay) {
			out.push(`UNTIL=${r.until.replaceAll('-', '')}`);
		} else {
			const iso = new Date(`${r.until}T23:59:59`).toISOString();
			out.push(`UNTIL=${iso.replace(/[-:]/g, '').replace(/\.\d{3}/, '')}`);
		}
	}
	return out.join(';');
}

const UNITS: Record<string, [string, string, string]> = {
	DAILY: ['Tous les jours', 'jours', 'Tous les'],
	WEEKLY: ['Toutes les semaines', 'semaines', 'Toutes les'],
	MONTHLY: ['Tous les mois', 'mois', 'Tous les'],
	YEARLY: ['Tous les ans', 'ans', 'Tous les']
};

/** Returns the words around an interval input, such as ["Toutes les", "semaines"]. */
export function intervalWords(freq: Frequency): [string, string] {
	const unit = UNITS[freq];
	return unit ? [unit[2], unit[1]] : ['', ''];
}

/** Describes an RRULE in French, such as "Toutes les semaines le lundi, 10 fois". */
export function describeRule(rule: string, start: Date): string {
	if (!rule) return '';
	const r = parseRule(rule, start);
	if (r.freq === 'custom' || r.freq === 'none') return 'Récurrence personnalisée';
	const [single, plural, lead] = UNITS[r.freq];
	let text = r.interval > 1 ? `${lead} ${r.interval} ${plural}` : single;
	if (r.freq === 'WEEKLY') {
		const days = WEEKDAYS.filter((d) => r.byDay.includes(d));
		if (r.interval === 1 && days.join() === 'MO,TU,WE,TH,FR') text = 'Du lundi au vendredi';
		else if (days.length === 7) text = r.interval === 1 ? 'Tous les jours' : text;
		else text += ` le ${days.map((d) => WEEKDAY_NAMES[WEEKDAYS.indexOf(d)]).join(', ')}`;
	}
	if (r.end === 'count') text += `, ${r.count} fois`;
	if (r.end === 'until' && r.until) {
		const until = new Date(`${r.until}T00:00`);
		text += `, jusqu'au ${until.toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' })}`;
	}
	return text;
}
