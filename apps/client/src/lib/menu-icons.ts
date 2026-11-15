import { icons, registerIcons } from '@facile/muse';
import data from './menu-icons.json';

registerIcons(data);

export const menuIcons = {
	add: 'solar:add-circle-linear',
	edit: 'solar:pen-linear',
	duplicate: icons.copy,
	day: 'solar:calendar-date-linear',
	undo: 'solar:undo-left-round-linear',
	redo: 'solar:undo-right-round-linear',
	trash: icons.remove
} as const;
