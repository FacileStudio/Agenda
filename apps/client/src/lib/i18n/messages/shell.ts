import type { Row } from '../row';

/** Strings shared by the whole app: buttons, the user menu, the sign-in and landing pages. */
export const shell = {
	'common.cancel': ['Cancel', 'Annuler', 'Cancelar', 'Abbrechen'],
	'common.save': ['Save', 'Enregistrer', 'Guardar', 'Speichern'],
	'common.saving': ['Saving…', 'Enregistrement…', 'Guardando…', 'Speichern…'],
	'common.create': ['Create', 'Créer', 'Crear', 'Erstellen'],
	'common.creating': ['Creating…', 'Création…', 'Creando…', 'Wird erstellt…'],
	'common.delete': ['Delete', 'Supprimer', 'Eliminar', 'Löschen'],
	'common.deleting': ['Deleting…', 'Suppression…', 'Eliminando…', 'Wird gelöscht…'],
	'common.remove': ['Remove', 'Retirer', 'Quitar', 'Entfernen'],
	'common.close': ['Close', 'Fermer', 'Cerrar', 'Schließen'],
	'common.revoke': ['Revoke', 'Révoquer', 'Revocar', 'Widerrufen'],
	'common.loading': ['Loading', 'Chargement', 'Cargando', 'Wird geladen'],
	'common.loadingDots': ['Loading…', 'Chargement…', 'Cargando…', 'Wird geladen…'],
	'common.error': ['Error.', 'Erreur.', 'Error.', 'Fehler.'],
	'common.genericError': [
		'Something went wrong.',
		'Une erreur est survenue.',
		'Se ha producido un error.',
		'Es ist ein Fehler aufgetreten.'
	],
	'common.open': ['Open', 'Ouvrir', 'Abrir', 'Öffnen'],
	'common.untitled': ['Untitled', 'Sans titre', 'Sin título', 'Ohne Titel'],
	'common.allDay': ['All day', 'Toute la journée', 'Todo el día', 'Ganztägig'],
	'common.today': ['Today', 'Aujourd’hui', 'Hoy', 'Heute'],
	'common.name': ['Name', 'Nom', 'Nombre', 'Name'],
	'common.email': ['Email', 'Email', 'Correo electrónico', 'E-Mail'],
	'common.description': ['Description', 'Description', 'Descripción', 'Beschreibung'],
	'common.color': ['Color', 'Couleur', 'Color', 'Farbe'],
	'common.members': ['Members', 'Membres', 'Miembros', 'Mitglieder'],
	'common.manageInRegistre': [
		'Manage in Registre',
		'Gérer dans Registre',
		'Gestionar en Registre',
		'In Registre verwalten'
	],
	'role.owner': ['Owner', 'Propriétaire', 'Propietario', 'Eigentümer'],
	'role.admin': ['Administrator', 'Administrateur', 'Administrador', 'Administrator'],
	'role.writer': ['Editor', 'Éditeur', 'Editor', 'Bearbeiter'],
	'role.reader': ['Reader', 'Lecteur', 'Lector', 'Leser'],
	'role.member': ['Member', 'Membre', 'Miembro', 'Mitglied'],
	'menu.profile': ['Profile menu', 'Menu du profil', 'Menú del perfil', 'Profilmenü'],
	'menu.myProfile': ['My profile', 'Mon profil', 'Mi perfil', 'Mein Profil'],
	'menu.calendar': ['Calendar', 'Calendrier', 'Calendario', 'Kalender'],
	'menu.settings': ['Settings', 'Paramètres', 'Ajustes', 'Einstellungen'],
	'menu.spaces': ['Spaces', 'Espaces', 'Espacios', 'Bereiche'],
	'menu.members': [
		'Calendars and members',
		'Calendriers et membres',
		'Calendarios y miembros',
		'Kalender und Mitglieder'
	],
	'menu.logout': ['Sign out', 'Se déconnecter', 'Cerrar sesión', 'Abmelden'],
	'menu.logoutUnconfirmed': [
		'The server did not confirm the sign-out; the session is closed here.',
		'Le serveur n’a pas confirmé la déconnexion, la session est fermée ici.',
		'El servidor no ha confirmado el cierre de sesión; la sesión se ha cerrado aquí.',
		'Der Server hat die Abmeldung nicht bestätigt; die Sitzung ist hier beendet.'
	],
	'login.title': ['Sign in — Agenda', 'Connexion — Agenda', 'Iniciar sesión — Agenda', 'Anmelden — Agenda'],
	'login.tagline': [
		'Your calendar. Your server.',
		'Votre agenda. Votre serveur.',
		'Tu calendario. Tu servidor.',
		'Dein Kalender. Dein Server.'
	],
	'login.failed': [
		'Sign-in failed. Try again.',
		'La connexion a échoué. Réessayez.',
		'No se pudo iniciar sesión. Inténtalo de nuevo.',
		'Die Anmeldung ist fehlgeschlagen. Versuche es erneut.'
	]
} satisfies Record<string, Row>;
