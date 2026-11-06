import type { Catalogs, Message } from '@facile/muse';
import { calendar } from './calendar';
import { event } from './event';
import { landing } from './landing';
import { recurrence } from './recurrence';
import { settings } from './settings';
import { settingsAdmin } from './settings-admin';
import { settingsApi } from './settings-api';
import { shell } from './shell';
import { spaces } from './spaces';
import type { Row } from '../row';

const rows = {
	...shell,
	...landing,
	...calendar,
	...event,
	...recurrence,
	...spaces,
	...settings,
	...settingsApi,
	...settingsAdmin
};

export type AppKey = keyof typeof rows;
type AppCatalog = Record<AppKey, Message>;

function column(index: number): AppCatalog {
	const entries = Object.entries(rows as Record<string, Row>).map(([key, row]) => [key, row[index]]);
	return Object.fromEntries(entries) as AppCatalog;
}

/** The app's catalogs: English is the typed base, and every row carries all four locales. */
export const catalogs: Catalogs<AppCatalog> = {
	en: column(0),
	fr: column(1),
	es: column(2),
	de: column(3)
};
