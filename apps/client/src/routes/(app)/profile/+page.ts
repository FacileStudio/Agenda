import { redirect } from '@sveltejs/kit';

export const prerender = false;

/**
 * The profile page was a second, thinner copy of the settings Profile tab — same avatar,
 * same two disabled fields, same SSO sync button. CHARTE §14 puts identity on the Profile
 * tab of the one settings page, so /profile now just points there. The redirect stays
 * because the URL is linked from the sidebar and the mobile nav.
 */
export function load() {
	redirect(307, '/settings');
}
