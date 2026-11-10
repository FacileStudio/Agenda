import type { Row } from '../row';

/** The page where the person confirms what an AI client asked to do. */
export const pending = {
	'pending.title': ['Pending actions', 'Actions en attente', 'Acciones pendientes', 'Ausstehende Aktionen'],
	'pending.intro': [
		'An AI client asked Agenda to do these. Nothing happens until you confirm.',
		'Un client IA a demandé ces actions à Agenda. Rien ne se passe tant que vous ne confirmez pas.',
		'Un cliente de IA pidió estas acciones a Agenda. No ocurre nada hasta que confirmes.',
		'Ein KI-Client hat Agenda um diese Aktionen gebeten. Nichts passiert, bevor du bestätigst.'
	],
	'pending.none': ['Nothing is waiting', 'Rien en attente', 'Nada pendiente', 'Nichts ausstehend'],
	'pending.noneBody': [
		'Actions an AI client asks for appear here until you confirm or dismiss them.',
		'Les actions demandées par un client IA apparaissent ici jusqu’à ce que vous les confirmiez ou les ignoriez.',
		'Las acciones que pide un cliente de IA aparecen aquí hasta que las confirmes o descartes.',
		'Aktionen, um die ein KI-Client bittet, erscheinen hier, bis du sie bestätigst oder verwirfst.'
	],
	'pending.deleteEvent': ['Delete event', 'Supprimer l’événement', 'Eliminar evento', 'Termin löschen'],
	'pending.askedBy': ['Asked by {client}', 'Demandé par {client}', 'Pedido por {client}', 'Angefragt von {client}'],
	'pending.recurring': [
		'Recurring: every occurrence goes',
		'Récurrent : toutes les occurrences seront supprimées',
		'Recurrente: se eliminan todas las repeticiones',
		'Wiederkehrend: alle Termine der Serie werden gelöscht'
	],
	'pending.confirmDelete': ['Delete', 'Supprimer', 'Eliminar', 'Löschen'],
	'pending.dismiss': ['Dismiss', 'Ignorer', 'Descartar', 'Verwerfen'],
	'pending.changed': [
		'The event changed after the request. Ask the AI client again.',
		'L’événement a changé après la demande. Redemandez au client IA.',
		'El evento cambió después de la solicitud. Vuelve a pedírselo al cliente de IA.',
		'Der Termin wurde nach der Anfrage geändert. Frag den KI-Client erneut.'
	],
	'pending.missing': [
		'The event no longer exists.',
		'L’événement n’existe plus.',
		'El evento ya no existe.',
		'Der Termin existiert nicht mehr.'
	],
	'pending.deleted': ['Event deleted', 'Événement supprimé', 'Evento eliminado', 'Termin gelöscht'],
	'pending.dismissed': [
		'Dismissed. The event is kept.',
		'Ignoré. L’événement est conservé.',
		'Descartado. Se conserva el evento.',
		'Verworfen. Der Termin bleibt erhalten.'
	],
	'pending.failed': ['That did not work', 'Cela n’a pas marché', 'No ha funcionado', 'Das hat nicht geklappt'],
	'pending.loadFailed': [
		'Could not load pending actions',
		'Impossible de charger les actions en attente',
		'No se pudieron cargar las acciones pendientes',
		'Ausstehende Aktionen konnten nicht geladen werden'
	]
} satisfies Record<string, Row>;
