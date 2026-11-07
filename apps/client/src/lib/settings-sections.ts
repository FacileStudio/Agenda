import { icons } from '@facile/muse';

export const SECTION_IDS = [
	'profile',
	'preferences',
	'notifications',
	'caldav',
	'members',
	'tokens',
	'advanced'
] as const;

export const LEGACY_SECTIONS: Record<string, string> = {
	appearance: 'preferences',
	api: 'caldav'
};

export const SECTION_ICONS: Record<(typeof SECTION_IDS)[number], string> = {
	profile: icons.userCircle,
	preferences: icons.palette,
	notifications: icons.notification,
	caldav: icons.server,
	members: icons.usersGroup,
	tokens: icons.key,
	advanced: icons.shield
};
