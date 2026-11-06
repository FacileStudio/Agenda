import type { Row } from '../row';

/** Strings of the calendar page, its four views and the calendar filter. */
export const calendar = {
	'view.month': ['Month', 'Mois', 'Mes', 'Monat'],
	'view.week': ['Week', 'Semaine', 'Semana', 'Woche'],
	'view.day': ['Day', 'Jour', 'Día', 'Tag'],
	'view.agenda': ['Agenda', 'Agenda', 'Agenda', 'Agenda'],
	'calendar.new': ['New', 'Nouveau', 'Nuevo', 'Neu'],
	'calendar.newEvent': ['New event', 'Nouvel événement', 'Nuevo evento', 'Neuer Termin'],
	'calendar.previous': ['Previous', 'Précédent', 'Anterior', 'Zurück'],
	'calendar.next': ['Next', 'Suivant', 'Siguiente', 'Weiter'],
	'calendar.backToToday': [
		'Back to today',
		'Revenir à aujourd’hui',
		'Volver a hoy',
		'Zurück zu heute'
	],
	'calendar.filter': [
		'Filter calendars',
		'Filtrer les calendriers',
		'Filtrar calendarios',
		'Kalender filtern'
	],
	'calendar.all': ['All calendars', 'Tous les calendriers', 'Todos los calendarios', 'Alle Kalender'],
	'calendar.space': ['Space {name}', 'Espace {name}', 'Espacio {name}', 'Bereich {name}'],
	'calendar.moreEvents': ['+{count} more', '+{count} de plus', '+{count} más', '+{count} weitere'],
	'calendar.createAt': [
		'Create an event at {time}',
		'Créer un événement à {time}',
		'Crear un evento a las {time}',
		'Termin um {time} erstellen'
	],
	'calendar.recurring': ['Recurring', 'Récurrent', 'Recurrente', 'Wiederkehrend'],
	'calendar.empty': [
		'No events to show',
		'Aucun événement à afficher',
		'No hay eventos que mostrar',
		'Keine Termine anzuzeigen'
	],
	'calendar.emptyHint': [
		'Click a time slot to create one.',
		'Cliquez sur un créneau pour en créer un.',
		'Haz clic en una franja para crear uno.',
		'Klicke auf einen Zeitraum, um einen zu erstellen.'
	],
	'calendar.createdBy': ['Created by {name}', 'Créé par {name}', 'Creado por {name}', 'Erstellt von {name}'],
	'events.title': ['Events', 'Événements', 'Eventos', 'Termine'],
	'events.subtitle': [
		'Your events for the next 30 days',
		'Vos événements des 30 prochains jours',
		'Tus eventos de los próximos 30 días',
		'Deine Termine der nächsten 30 Tage'
	],
	'events.empty': [
		'No upcoming events',
		'Aucun événement à venir',
		'No hay eventos próximos',
		'Keine bevorstehenden Termine'
	],
	'events.emptyHint': [
		'Your appointments for the next 30 days will show up here.',
		'Vos rendez-vous des 30 prochains jours apparaîtront ici.',
		'Tus citas de los próximos 30 días aparecerán aquí.',
		'Deine Termine der nächsten 30 Tage erscheinen hier.'
	],
	'calendarModal.newTitle': ['New calendar', 'Nouveau calendrier', 'Nuevo calendario', 'Neuer Kalender'],
	'calendarModal.manageTitle': [
		'Manage the calendar',
		'Gérer le calendrier',
		'Gestionar el calendario',
		'Kalender verwalten'
	],
	'calendarModal.namePlaceholder': ['My calendar', 'Mon calendrier', 'Mi calendario', 'Mein Kalender'],
	'calendarModal.nameLong': ['Calendar name', 'Nom du calendrier', 'Nombre del calendario', 'Kalendername'],
	'calendarModal.colorLabel': [
		'Calendar color',
		'Couleur du calendrier',
		'Color del calendario',
		'Kalenderfarbe'
	],
	'calendarModal.descriptionPlaceholder': [
		'Description (optional)',
		'Description (optionnelle)',
		'Descripción (opcional)',
		'Beschreibung (optional)'
	],
	'calendarModal.echo': [
		'Echo (video call)',
		'Echo (visioconférence)',
		'Echo (videollamada)',
		'Echo (Videokonferenz)'
	],
	'calendarModal.echoHelper': [
		'URL of the Echo instance for this calendar',
		'URL de l’instance Echo pour ce calendrier',
		'URL de la instancia de Echo para este calendario',
		'URL der Echo-Instanz für diesen Kalender'
	],
	'calendarModal.nameRequired': [
		'The name is required.',
		'Le nom est requis.',
		'El nombre es obligatorio.',
		'Der Name ist erforderlich.'
	],
	'calendarModal.deleteTitle': [
		'Delete this calendar?',
		'Supprimer ce calendrier ?',
		'¿Eliminar este calendario?',
		'Diesen Kalender löschen?'
	],
	'calendarModal.deleteText': [
		'“{name}” and all its events will be deleted for every member. This cannot be undone.',
		'« {name} » et tous ses événements seront supprimés pour chaque membre. Cette action est irréversible.',
		'«{name}» y todos sus eventos se eliminarán para todos los miembros. Esta acción no se puede deshacer.',
		'„{name}“ und alle zugehörigen Termine werden für alle Mitglieder gelöscht. Das lässt sich nicht rückgängig machen.'
	],
	'calendarModal.dangerZone': ['Danger zone', 'Zone dangereuse', 'Zona de peligro', 'Gefahrenzone'],
	'calendarModal.deleteButton': [
		'Delete this calendar',
		'Supprimer ce calendrier',
		'Eliminar este calendario',
		'Diesen Kalender löschen'
	],
	'member.none': [
		'No members shared yet.',
		'Aucun membre partagé.',
		'Aún no hay miembros compartidos.',
		'Noch keine Mitglieder geteilt.'
	],
	'member.invite': ['Invite a member', 'Inviter un membre', 'Invitar a un miembro', 'Mitglied einladen'],
	'member.inviteButton': ['Invite', 'Inviter', 'Invitar', 'Einladen'],
	'member.emailAria': [
		'Email of the member to invite',
		'Email du membre à inviter',
		'Correo del miembro a invitar',
		'E-Mail des einzuladenden Mitglieds'
	],
	'member.emailPlaceholder': ['email@example.com', 'email@exemple.com', 'correo@ejemplo.com', 'name@beispiel.de'],
	'member.roleAria': ['Role', 'Rôle', 'Rol', 'Rolle'],
	'member.emailRequired': ['Email required.', 'Email requis.', 'Correo obligatorio.', 'E-Mail erforderlich.'],
	'member.emailInvalid': [
		'Invalid email address.',
		'Adresse email invalide.',
		'Dirección de correo no válida.',
		'Ungültige E-Mail-Adresse.'
	],
	'member.removeAria': ['Remove {name}', 'Retirer {name}', 'Quitar a {name}', '{name} entfernen'],
	'member.removeTitle': [
		'Remove this member?',
		'Retirer ce membre ?',
		'¿Quitar a este miembro?',
		'Dieses Mitglied entfernen?'
	],
	'member.removeText': [
		'{name} will immediately lose access to this calendar.',
		'{name} perdra immédiatement l’accès à ce calendrier.',
		'{name} perderá el acceso a este calendario de inmediato.',
		'{name} verliert sofort den Zugriff auf diesen Kalender.'
	],
	'member.thisPerson': ['This person', 'Cette personne', 'Esta persona', 'Diese Person'],
	'member.err404': [
		'No account found with this email. The person must sign in to Agenda first.',
		'Aucun compte trouvé avec cet email. La personne doit d’abord se connecter à Agenda.',
		'No se encontró ninguna cuenta con este correo. La persona debe iniciar sesión primero en Agenda.',
		'Kein Konto mit dieser E-Mail gefunden. Die Person muss sich zuerst bei Agenda anmelden.'
	],
	'member.err403': [
		'You are not allowed to share this calendar.',
		'Vous n’avez pas la permission de partager ce calendrier.',
		'No tienes permiso para compartir este calendario.',
		'Du darfst diesen Kalender nicht teilen.'
	],
	'member.errSelf': [
		'You cannot invite yourself.',
		'Vous ne pouvez pas vous inviter vous-même.',
		'No puedes invitarte a ti mismo.',
		'Du kannst dich nicht selbst einladen.'
	],
	'member.err400': [
		'Invalid invitation. Check the email and the role.',
		'Invitation invalide. Vérifiez l’email et le rôle.',
		'Invitación no válida. Comprueba el correo y el rol.',
		'Ungültige Einladung. Prüfe die E-Mail und die Rolle.'
	],
	'member.errFailed': [
		'The invitation failed. Try again.',
		'L’invitation a échoué. Réessayez.',
		'La invitación ha fallado. Inténtalo de nuevo.',
		'Die Einladung ist fehlgeschlagen. Versuche es erneut.'
	]
} satisfies Record<string, Row>;
