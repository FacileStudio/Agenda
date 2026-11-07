import type { Row } from '../row';

/** Strings of the settings pages other than API. */
export const settings = {
	'settings.title': ['Settings', 'Paramètres', 'Ajustes', 'Einstellungen'],
	'settings.subtitle': [
		'Your account, this browser, and how your CalDAV clients reach Agenda.',
		'Votre compte, ce navigateur, et la façon dont vos clients CalDAV joignent Agenda.',
		'Tu cuenta, este navegador y la forma en que tus clientes CalDAV llegan a Agenda.',
		'Dein Konto, dieser Browser und wie deine CalDAV-Clients Agenda erreichen.'
	],
	'settings.tab.profile': ['Profile', 'Profil', 'Perfil', 'Profil'],
	'settings.tab.preferences': ['Preferences', 'Préférences', 'Preferencias', 'Voreinstellungen'],
	'settings.tab.notifications': ['Notifications', 'Notifications', 'Notificaciones', 'Benachrichtigungen'],
	'settings.tab.caldav': ['CalDAV', 'CalDAV', 'CalDAV', 'CalDAV'],
	'settings.tab.tokens': ['API tokens', 'Tokens API', 'Tokens de API', 'API-Tokens'],
	'settings.tab.members': ['Members', 'Membres', 'Miembros', 'Mitglieder'],
	'settings.tab.advanced': ['Advanced', 'Avancé', 'Avanzado', 'Erweitert'],
	'profile.sync': ['Sync SSO profile', 'Sync. profil SSO', 'Sincronizar perfil SSO', 'SSO-Profil synchronisieren'],
	'profile.syncing': ['Syncing…', 'Synchronisation…', 'Sincronizando…', 'Synchronisiere…'],
	'profile.synced': [
		'Profile synced from SSO.',
		'Profil synchronisé depuis SSO.',
		'Perfil sincronizado desde SSO.',
		'Profil aus SSO synchronisiert.'
	],
	'profile.upToDate': [
		'Profile already up to date.',
		'Profil déjà à jour.',
		'El perfil ya está al día.',
		'Profil bereits aktuell.'
	],
	'profile.syncFailed': [
		'Sync failed.',
		'Échec de la synchronisation.',
		'Error de sincronización.',
		'Synchronisierung fehlgeschlagen.'
	],
	'notifications.reminders': ['Reminders', 'Rappels', 'Recordatorios', 'Erinnerungen'],
	'notifications.remindersText': [
		'Agenda does not notify by itself. It publishes your events, and your client rings.',
		'Agenda ne notifie pas lui-même. Il publie vos événements, et c’est votre client qui sonne.',
		'Agenda no notifica por sí mismo. Publica tus eventos y es tu cliente el que suena.',
		'Agenda benachrichtigt nicht selbst. Es veröffentlicht deine Termine, und dein Client klingelt.'
	],
	'notifications.alert': [
		'A reminder travels inside the event, not beside it: Agenda keeps the original ICS and serves it unchanged over CalDAV, alarms included. Set your reminders in iOS Calendar, macOS Calendar, Thunderbird or DAVx⁵; they come back here and go out to your other devices.',
		'Un rappel voyage dans l’événement, pas à côté : Agenda conserve l’ICS d’origine et le rend tel quel en CalDAV, alarmes comprises. Réglez donc vos rappels dans Calendrier iOS, Calendrier macOS, Thunderbird ou DAVx⁵ : ils remontent ici et repartent vers vos autres appareils.',
		'Un recordatorio viaja dentro del evento, no al lado: Agenda conserva el ICS original y lo sirve tal cual por CalDAV, alarmas incluidas. Configura tus recordatorios en Calendario de iOS, Calendario de macOS, Thunderbird o DAVx⁵; suben aquí y salen hacia tus otros dispositivos.',
		'Eine Erinnerung steckt im Termin, nicht daneben: Agenda behält das Original-ICS und liefert es unverändert über CalDAV aus, Alarme eingeschlossen. Stelle deine Erinnerungen in iOS Kalender, macOS Kalender, Thunderbird oder DAVx⁵ ein; sie kommen hierher zurück und gehen an deine anderen Geräte.'
	],
	'notifications.client': [
		'Reminders in the client',
		'Rappels dans le client',
		'Recordatorios en el cliente',
		'Erinnerungen im Client'
	],
	'notifications.clientText': [
		'The VALARMs of your events are kept byte for byte through CalDAV round trips.',
		'Les VALARM de vos événements sont conservés à l’octet près lors des allers-retours CalDAV.',
		'Los VALARM de tus eventos se conservan byte a byte en los viajes de ida y vuelta de CalDAV.',
		'Die VALARMs deiner Termine bleiben bei CalDAV-Hin-und-zurück-Übertragungen bytegenau erhalten.'
	],
	'notifications.supported': ['Supported', 'Pris en charge', 'Compatible', 'Unterstützt'],
	'notifications.emails': ['Emails', 'E-mails', 'Correos', 'E-Mails'],
	'notifications.emailsText': [
		'This instance has no SMTP relay configured: no invitation or reminder is sent by email.',
		'Cette instance n’a pas de relais SMTP configuré : aucune invitation ni aucun rappel n’est envoyé par mail.',
		'Esta instancia no tiene un relé SMTP configurado: no se envía ninguna invitación ni recordatorio por correo.',
		'Diese Instanz hat kein SMTP-Relay konfiguriert: Einladungen und Erinnerungen werden nicht per E-Mail versendet.'
	],
	'notifications.none': ['Nothing sent', 'Aucun envoi', 'Sin envíos', 'Kein Versand'],
	'notifications.browser': [
		'Browser notifications',
		'Notifications navigateur',
		'Notificaciones del navegador',
		'Browser-Benachrichtigungen'
	],
	'notifications.browserText': [
		'Agenda does not ask for the Notifications permission and registers no push service worker.',
		'Agenda ne demande pas la permission Notifications et n’enregistre pas de service worker push.',
		'Agenda no solicita el permiso de Notificaciones ni registra un service worker de push.',
		'Agenda fragt die Benachrichtigungsberechtigung nicht an und registriert keinen Push-Service-Worker.'
	],
	'notifications.notImplemented': ['Not implemented', 'Non implémenté', 'No implementado', 'Nicht umgesetzt'],
	'notifications.where': [
		'Where to set reminders',
		'Où régler ses rappels',
		'Dónde configurar los recordatorios',
		'Wo du Erinnerungen einstellst'
	],
	'notifications.whereText': [
		'Connect a CalDAV client first, then set the alarms in it.',
		'Connectez d’abord un client CalDAV, puis réglez les alarmes dedans.',
		'Conecta primero un cliente CalDAV y configura las alarmas en él.',
		'Verbinde zuerst einen CalDAV-Client und stelle die Alarme dort ein.'
	],
	'notifications.caldavButton': ['CalDAV connection', 'Connexion CalDAV', 'Conexión CalDAV', 'CalDAV-Verbindung'],
	'notifications.sharing': ['Sharing an event', 'Partage d’événement', 'Compartir un evento', 'Termin teilen'],
	'notifications.sharingText': [
		'An event’s attendees are stored, but they receive no invitation: share the calendar rather than the event.',
		'Les participants d’un événement sont enregistrés, mais ils ne reçoivent pas d’invitation : partagez le calendrier plutôt que l’événement.',
		'Los asistentes de un evento se guardan, pero no reciben ninguna invitación: comparte el calendario en lugar del evento.',
		'Die Teilnehmer eines Termins werden gespeichert, erhalten aber keine Einladung: teile den Kalender statt des Termins.'
	],
	'notifications.sharingButton': [
		'Calendar sharing',
		'Partage des calendriers',
		'Compartir calendarios',
		'Kalenderfreigabe'
	]
} satisfies Record<string, Row>;
