import { registerIcons } from '@facile/muse';
import data from './landing-icons.json';

/** The Solar glyphs the landing draws that muse does not bundle, so none of them waits on the network. */
export const LANDING_ICONS = {
	agenda: 'solar:calendar-add-bold-duotone',
	opus: 'solar:slider-vertical-minimalistic-bold-duotone',
	bulle: 'solar:smile-circle-bold-duotone',
	phone: 'solar:smartphone-2-linear',
	laptop: 'solar:laptop-minimalistic-linear',
	synced: 'solar:check-circle-bold',
	week: 'solar:calendar-minimalistic-linear',
	day: 'solar:calendar-date-linear',
	list: 'solar:list-linear'
} as const;

registerIcons(data);
