import type { Row } from '../row';

/** Strings of the public landing page. */
export const landing = {
	'landing.heroTitle': [
		'Your calendar.|Your server.',
		'Votre agenda.|Votre serveur.',
		'Tu calendario.|Tu servidor.',
		'Dein Kalender.|Dein Server.'
	],
	'landing.heroText': [
		'Plan the studio week together and sync it to every device.',
		'Planifiez la semaine du studio ensemble et synchronisez-la sur tous vos appareils.',
		'Planifica la semana del estudio en equipo y sincronízala en todos tus dispositivos.',
		'Plane die Studiowoche gemeinsam und synchronisiere sie auf jedem Gerät.'
	],
	'landing.getCli': ['Get the CLI', 'Installer le CLI', 'Instalar la CLI', 'CLI installieren'],
	'landing.feature.caldav.title': ['CalDAV sync', 'Synchro CalDAV', 'Sincronización CalDAV', 'CalDAV-Sync'],
	'landing.feature.caldav.text': [
		'Apple Calendar, Thunderbird and DAVx⁵ connect natively, with no plugin.',
		'Calendrier Apple, Thunderbird et DAVx⁵ se connectent nativement, sans plugin.',
		'Calendario de Apple, Thunderbird y DAVx⁵ se conectan de forma nativa, sin plugins.',
		'Apple Kalender, Thunderbird und DAVx⁵ verbinden sich nativ, ohne Plugin.'
	],
	'landing.feature.shared.title': [
		'One calendar per space',
		'Un agenda par espace',
		'Un calendario por espacio',
		'Ein Kalender pro Space'
	],
	'landing.feature.shared.text': [
		'Every member of a Facile space reads and edits the same calendar.',
		'Chaque membre d’un espace Facile lit et modifie le même agenda.',
		'Cada miembro de un espacio Facile lee y edita el mismo calendario.',
		'Jedes Mitglied eines Facile-Space liest und bearbeitet denselben Kalender.'
	],
	'landing.feature.hosted.title': ['Self-hosted', 'Auto-hébergé', 'Autoalojado', 'Selbst gehostet'],
	'landing.feature.hosted.text': [
		'One binary and a Postgres. Your events stay on your server.',
		'Un binaire et un Postgres. Vos événements restent sur votre serveur.',
		'Un binario y un Postgres. Tus eventos se quedan en tu servidor.',
		'Eine Binärdatei und ein Postgres. Deine Termine bleiben auf deinem Server.'
	],
	'landing.feature.cli.title': ['From the terminal', 'Depuis le terminal', 'Desde la terminal', 'Im Terminal'],
	'landing.feature.cli.text': [
		'Create and edit events with the agenda CLI or the API.',
		'Créez et modifiez vos événements avec le CLI agenda ou l’API.',
		'Crea y edita eventos con la CLI agenda o la API.',
		'Lege Termine mit der agenda-CLI oder der API an und bearbeite sie.'
	],
	'landing.download.title': ['Agenda in your terminal', 'Agenda dans votre terminal', 'Agenda en tu terminal', 'Agenda im Terminal'],
	'landing.download.text': [
		'Install the CLI, then run agenda login.',
		'Installez le CLI, puis lancez agenda login.',
		'Instala la CLI y ejecuta agenda login.',
		'Installiere die CLI und führe agenda login aus.'
	],
	'landing.download.facile': ['Facile installer', 'Installeur Facile', 'Instalador de Facile', 'Facile-Installer'],
	'landing.download.script': ['Install script', 'Script d’installation', 'Script de instalación', 'Installationsskript'],
	'landing.demo.label': [
		'A new event is added to the week view, then syncs over CalDAV.',
		'Un nouvel événement est ajouté à la vue semaine, puis synchronisé en CalDAV.',
		'Se añade un evento a la vista semanal y se sincroniza por CalDAV.',
		'Ein neuer Termin landet in der Wochenansicht und wird per CalDAV synchronisiert.'
	],
	'landing.demo.event': ['Studio review', 'Revue du studio', 'Revisión del estudio', 'Studio-Review'],
	'landing.demo.syncing': ['Syncing', 'Synchronisation', 'Sincronizando', 'Wird synchronisiert'],
	'landing.demo.synced': ['Synced', 'Synchronisé', 'Sincronizado', 'Synchronisiert'],
	'landing.ctaTitle': ['Ready to plan the week?', 'Prêt à planifier la semaine ?', '¿Listo para planificar la semana?', 'Bereit, die Woche zu planen?'],
	'landing.ctaText': [
		'Sign in with your Facile account.',
		'Connectez-vous avec votre compte Facile.',
		'Inicia sesión con tu cuenta de Facile.',
		'Melde dich mit deinem Facile-Konto an.'
	],
	'landing.pricing': ['Pricing', 'Tarifs', 'Precios', 'Preise']
} satisfies Record<string, Row>;
