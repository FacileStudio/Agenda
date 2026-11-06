import type { Row } from '../row';

/** Strings of the public landing page. */
export const landing = {
	'landing.title': [
		'Agenda — Self-hosted calendar for creative studios',
		'Agenda — Calendrier auto-hébergé pour studios créatifs',
		'Agenda — Calendario autoalojado para estudios creativos',
		'Agenda — Selbst gehosteter Kalender für Kreativstudios'
	],
	'landing.description': [
		'A self-hosted calendar for creative studios. CalDAV sync, shared calendars, no cloud.',
		'Un calendrier auto-hébergé pour studios créatifs. Synchro CalDAV, calendriers partagés, pas de cloud.',
		'Un calendario autoalojado para estudios creativos. Sincronización CalDAV, calendarios compartidos, sin nube.',
		'Ein selbst gehosteter Kalender für Kreativstudios. CalDAV-Sync, geteilte Kalender, keine Cloud.'
	],
	'landing.heroTitle': [
		'Your calendar.|Your server.',
		'Votre agenda.|Votre serveur.',
		'Tu calendario.|Tu servidor.',
		'Dein Kalender.|Dein Server.'
	],
	'landing.heroText': [
		'Agenda is a self-hosted calendar for creative studios. Plan the week, share it with the team, and sync it to every device you already use.',
		'Agenda est un calendrier auto-hébergé pour studios créatifs. Planifiez la semaine, partagez-la avec l’équipe et synchronisez-la sur tous vos appareils.',
		'Agenda es un calendario autoalojado para estudios creativos. Planifica la semana, compártela con el equipo y sincronízala con todos tus dispositivos.',
		'Agenda ist ein selbst gehosteter Kalender für Kreativstudios. Plane die Woche, teile sie mit dem Team und synchronisiere sie mit allen deinen Geräten.'
	],
	'landing.feature.caldav.title': ['CalDAV sync', 'Synchro CalDAV', 'Sincronización CalDAV', 'CalDAV-Sync'],
	'landing.feature.caldav.text': [
		'Apple Calendar, iOS, Thunderbird and DAVx⁵ all talk to it natively. No plugin, no export.',
		'Calendrier Apple, iOS, Thunderbird et DAVx⁵ le comprennent nativement. Ni plugin, ni export.',
		'Calendario de Apple, iOS, Thunderbird y DAVx⁵ lo entienden de forma nativa. Sin plugins ni exportaciones.',
		'Apple Kalender, iOS, Thunderbird und DAVx⁵ sprechen es nativ. Kein Plugin, kein Export.'
	],
	'landing.feature.shared.title': [
		'Shared calendars',
		'Calendriers partagés',
		'Calendarios compartidos',
		'Geteilte Kalender'
	],
	'landing.feature.shared.text': [
		'Invite your studio, set who can read and who can edit, keep personal calendars personal.',
		'Invitez votre studio, choisissez qui lit et qui modifie, gardez les calendriers personnels privés.',
		'Invita a tu estudio, decide quién puede leer y quién editar, y mantén privados los calendarios personales.',
		'Lade dein Studio ein, lege fest, wer lesen und wer bearbeiten darf, und halte persönliche Kalender privat.'
	],
	'landing.feature.hosted.title': ['Self-hosted', 'Auto-hébergé', 'Autoalojado', 'Selbst gehostet'],
	'landing.feature.hosted.text': [
		'One binary and a Postgres. Your events stay on your server, with no cloud and no tracking.',
		'Un binaire et un Postgres. Vos événements restent sur votre serveur, sans cloud ni pistage.',
		'Un binario y un Postgres. Tus eventos se quedan en tu servidor, sin nube ni rastreo.',
		'Eine Binärdatei und ein Postgres. Deine Termine bleiben auf deinem Server, ohne Cloud und ohne Tracking.'
	],
	'landing.ctaTitle': ['Ready to sign in?', 'Prêt à vous connecter ?', '¿Listo para iniciar sesión?', 'Bereit zur Anmeldung?'],
	'landing.ctaText': [
		'Use your Facile account to access Agenda.',
		'Utilisez votre compte Facile pour accéder à Agenda.',
		'Usa tu cuenta de Facile para acceder a Agenda.',
		'Nutze dein Facile-Konto, um auf Agenda zuzugreifen.'
	],
	'landing.pricing': ['Pricing', 'Tarifs', 'Precios', 'Preise']
} satisfies Record<string, Row>;
