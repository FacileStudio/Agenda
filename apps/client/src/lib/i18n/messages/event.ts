import type { Row } from '../row';

/** Strings of the event form and of the recurrence field. */
export const event = {
	'event.editTitle': ['Edit event', 'Modifier l’événement', 'Editar evento', 'Termin bearbeiten'],
	'event.newTitle': ['New event', 'Nouvel événement', 'Nuevo evento', 'Neuer Termin'],
	'event.title': ['Title', 'Titre', 'Título', 'Titel'],
	'event.titlePlaceholder': ['Event title', 'Titre de l’événement', 'Título del evento', 'Titel des Termins'],
	'event.calendar': ['Calendar', 'Calendrier', 'Calendario', 'Kalender'],
	'event.groupSpaces': ['Spaces', 'Espaces', 'Espacios', 'Bereiche'],
	'event.groupPersonal': ['Personal', 'Personnels', 'Personales', 'Persönlich'],
	'event.allDay': ['All day', 'Toute la journée', 'Todo el día', 'Ganztägig'],
	'event.start': ['Start', 'Début', 'Inicio', 'Beginn'],
	'event.end': ['End', 'Fin', 'Fin', 'Ende'],
	'event.startTime': ['Start time', 'Heure de début', 'Hora de inicio', 'Startzeit'],
	'event.endTime': ['End time', 'Heure de fin', 'Hora de fin', 'Endzeit'],
	'event.location': ['Location', 'Lieu', 'Lugar', 'Ort'],
	'event.locationPlaceholder': ['Location (optional)', 'Lieu (optionnel)', 'Lugar (opcional)', 'Ort (optional)'],
	'event.video': ['Video call', 'Visioconférence', 'Videollamada', 'Videokonferenz'],
	'event.videoAdd': ['Add an Echo call', 'Ajouter une visio Echo', 'Añadir una videollamada Echo', 'Echo-Videokonferenz hinzufügen'],
	'event.videoRemove': [
		'Remove the video call',
		'Retirer la visioconférence',
		'Quitar la videollamada',
		'Videokonferenz entfernen'
	],
	'event.description': ['Description', 'Description', 'Descripción', 'Beschreibung'],
	'event.descriptionPlaceholder': [
		'Description (optional)',
		'Description (optionnelle)',
		'Descripción (opcional)',
		'Beschreibung (optional)'
	],
	'event.status': ['Status', 'Statut', 'Estado', 'Status'],
	'event.confirmed': ['Confirmed', 'Confirmé', 'Confirmado', 'Bestätigt'],
	'event.tentative': ['Tentative', 'Provisoire', 'Provisional', 'Vorläufig'],
	'event.cancelled': ['Cancelled', 'Annulé', 'Cancelado', 'Abgesagt'],
	'event.createdBy': ['Created by {name}', 'Créé par {name}', 'Creado por {name}', 'Erstellt von {name}'],
	'event.seriesNote': [
		'Changes apply to every occurrence of the series.',
		'Les modifications s’appliquent à toutes les occurrences de la série.',
		'Los cambios se aplican a todas las repeticiones de la serie.',
		'Änderungen gelten für alle Termine der Serie.'
	],
	'event.titleRequired': [
		'The title is required.',
		'Le titre est requis.',
		'El título es obligatorio.',
		'Der Titel ist erforderlich.'
	],
	'event.calendarInvalid': [
		'Select a valid calendar.',
		'Sélectionnez un calendrier valide.',
		'Selecciona un calendario válido.',
		'Wähle einen gültigen Kalender.'
	],
	'event.endBeforeStart': [
		'The end must be after the start.',
		'La fin doit être après le début.',
		'El fin debe ser posterior al inicio.',
		'Das Ende muss nach dem Beginn liegen.'
	],
	'event.deleteSeriesTitle': [
		'Delete the whole series?',
		'Supprimer toute la série ?',
		'¿Eliminar toda la serie?',
		'Die ganze Serie löschen?'
	],
	'event.deleteTitle': [
		'Delete this event?',
		'Supprimer cet événement ?',
		'¿Eliminar este evento?',
		'Diesen Termin löschen?'
	],
	'event.thisEvent': ['this event', 'cet événement', 'este evento', 'diesen Termin'],
	'event.deleteSeriesText': [
		'Every occurrence of “{title}” will disappear from this calendar for everyone, and from CalDAV clients that already synced it. This cannot be undone.',
		'Toutes les occurrences de « {title} » disparaîtront de ce calendrier pour tout le monde, et des clients CalDAV déjà synchronisés. Cette action est irréversible.',
		'Todas las repeticiones de «{title}» desaparecerán de este calendario para todos y de los clientes CalDAV ya sincronizados. Esta acción no se puede deshacer.',
		'Alle Termine von „{title}“ verschwinden für alle aus diesem Kalender und aus bereits synchronisierten CalDAV-Clients. Das lässt sich nicht rückgängig machen.'
	],
	'event.deleteText': [
		'“{title}” will disappear from this calendar for everyone, and from CalDAV clients that already synced it. This cannot be undone.',
		'« {title} » disparaîtra de ce calendrier pour tout le monde, et des clients CalDAV déjà synchronisés. Cette action est irréversible.',
		'«{title}» desaparecerá de este calendario para todos y de los clientes CalDAV ya sincronizados. Esta acción no se puede deshacer.',
		'„{title}“ verschwindet für alle aus diesem Kalender und aus bereits synchronisierten CalDAV-Clients. Das lässt sich nicht rückgängig machen.'
	]
} satisfies Record<string, Row>;
