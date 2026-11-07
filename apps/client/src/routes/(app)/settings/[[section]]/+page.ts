import { redirect } from '@sveltejs/kit';
import { settingsRoute } from '@facile/muse';
import { SECTION_IDS, LEGACY_SECTIONS } from '$lib/settings-sections';

export const prerender = false;

export function load({ params }) {
	const requested = params.section;
	if (requested && requested in LEGACY_SECTIONS) redirect(307, `/settings/${LEGACY_SECTIONS[requested]}`);
	const route = settingsRoute(requested, SECTION_IDS.map((id) => ({ id })));
	if (!route.known) redirect(307, '/settings');
	return { section: requested };
}
