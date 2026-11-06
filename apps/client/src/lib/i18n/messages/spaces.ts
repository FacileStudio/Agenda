import type { Row } from '../row';

/** Strings of the spaces pages. */
export const spaces = {
	'spaces.title': ['Spaces', 'Espaces', 'Espacios', 'Bereiche'],
	'spaces.subtitle': [
		'Your shared spaces, the same in every Facile app. They are created and managed in Registre.',
		'Vos espaces partagés, les mêmes dans toutes les apps Facile. Ils se créent et se gèrent dans Registre.',
		'Tus espacios compartidos, los mismos en todas las apps de Facile. Se crean y se gestionan en Registre.',
		'Deine geteilten Bereiche, in allen Facile-Apps dieselben. Sie werden in Registre erstellt und verwaltet.'
	],
	'spaces.empty': [
		'No spaces yet.',
		'Aucun espace pour le moment.',
		'Todavía no hay espacios.',
		'Noch keine Bereiche.'
	],
	'spaces.create': [
		'Create a space in Registre',
		'Créer un espace dans Registre',
		'Crear un espacio en Registre',
		'Bereich in Registre erstellen'
	],
	'spaces.fallbackTitle': ['Space', 'Espace', 'Espacio', 'Bereich'],
	'spaces.back': ['Back to spaces', 'Retour aux espaces', 'Volver a los espacios', 'Zurück zu den Bereichen'],
	'spaces.viewCalendar': ['View the calendar', 'Voir le calendrier', 'Ver el calendario', 'Kalender anzeigen'],
	'spaces.viewInRegistre': ['View in Registre', 'Voir dans Registre', 'Ver en Registre', 'In Registre ansehen'],
	'spaces.info': ['Information', 'Informations', 'Información', 'Informationen'],
	'spaces.role': ['Role', 'Rôle', 'Rol', 'Rolle'],
	'spaces.createdOn': ['Created on', 'Créé le', 'Creado el', 'Erstellt am'],
	'spaces.inviteInRegistre': [
		'Invite or remove in Registre',
		'Inviter ou retirer dans Registre',
		'Invitar o quitar en Registre',
		'In Registre einladen oder entfernen'
	]
} satisfies Record<string, Row>;
