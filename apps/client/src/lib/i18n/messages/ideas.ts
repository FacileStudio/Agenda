import type { Row } from '../row';

/** Bulle run strings: tool names, approvals and errors. */
export const ideas = {
	'ideas.approval.summary': [
		'Bulle wants to run "{tool}". Check it before it goes ahead.',
		'Bulle veut lancer « {tool} ». Vérifiez avant de continuer.',
		'Bulle quiere ejecutar «{tool}». Revísalo antes de seguir.',
		'Bulle möchte „{tool}“ ausführen. Prüfen Sie es, bevor es weitergeht.'
	],
	'ideas.tool.agenda_create_event': ['Create an event', 'Créer un événement', 'Crear un evento', 'Termin anlegen'],
	'ideas.tool.agenda_update_event': ['Change an event', 'Modifier un événement', 'Modificar un evento', 'Termin ändern'],
	'ideas.tool.agenda_delete_event': ['Delete an event', 'Supprimer un événement', 'Eliminar un evento', 'Termin löschen'],
	'ideas.tool.agenda_read_event': ['Read an event', 'Lire un événement', 'Leer un evento', 'Termin lesen'],
	'ideas.tool.agenda_list_events': ['List events', 'Lister les événements', 'Listar eventos', 'Termine auflisten'],
	'ideas.tool.agenda_list_calendars': ['List calendars', 'Lister les agendas', 'Listar agendas', 'Kalender auflisten'],
	'ideas.tool.agenda_find_free_slots': ['Find free slots', 'Chercher des créneaux libres', 'Buscar huecos libres', 'Freie Zeiten suchen'],
	'ideas.error.load': [
		'Could not load the ideas.',
		'Impossible de charger les idées.',
		'No se pudieron cargar las ideas.',
		'Die Ideen konnten nicht geladen werden.'
	],
	'ideas.error.unreachable': [
		'Bulle is unreachable for now. Try again in a moment.',
		'Bulle est injoignable pour le moment. Réessayez dans un instant.',
		'Bulle no responde por ahora. Inténtalo en un momento.',
		'Bulle ist gerade nicht erreichbar. Versuchen Sie es gleich noch einmal.'
	],
	'ideas.error.notLinked': [
		'Reconnect your Facile account to use Bulle, or pick a space that Registre manages.',
		'Reconnectez votre compte Facile pour utiliser Bulle, ou choisissez un espace géré par Registre.',
		'Vuelve a conectar tu cuenta Facile para usar Bulle, o elige un espacio gestionado por Registre.',
		'Verbinden Sie Ihr Facile-Konto erneut oder wählen Sie einen von Registre verwalteten Bereich.'
	],
	'ideas.error.capped': [
		'You asked for ideas three times today. More come tomorrow.',
		'Vous avez demandé des idées trois fois aujourd’hui. D’autres arrivent demain.',
		'Ya pediste ideas tres veces hoy. Mañana llegan más.',
		'Sie haben heute dreimal nach Ideen gefragt. Morgen kommen neue.'
	],
	'ideas.error.busy': [
		'Bulle is already thinking of ideas for this space.',
		'Bulle cherche déjà des idées pour cet espace.',
		'Bulle ya está buscando ideas para este espacio.',
		'Bulle sucht bereits Ideen für diesen Bereich.'
	],
	'ideas.error.gone': [
		'This idea is no longer available.',
		'Cette idée n’est plus disponible.',
		'Esta idea ya no está disponible.',
		'Diese Idee ist nicht mehr verfügbar.'
	],
	'ideas.error.run': [
		'The run stopped before the end.',
		'L’exécution s’est arrêtée avant la fin.',
		'La ejecución se detuvo antes del final.',
		'Der Lauf wurde vor dem Ende abgebrochen.'
	]
} as const satisfies Record<string, Row>;
