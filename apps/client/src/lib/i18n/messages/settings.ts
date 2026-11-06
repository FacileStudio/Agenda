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
	'settings.sections': [
		'Settings sections',
		'Sections des paramètres',
		'Secciones de ajustes',
		'Einstellungsbereiche'
	],
	'settings.tab.profile': ['Profile', 'Profil', 'Perfil', 'Profil'],
	'settings.tab.appearance': ['Appearance', 'Apparence', 'Apariencia', 'Darstellung'],
	'settings.tab.notifications': ['Notifications', 'Notifications', 'Notificaciones', 'Benachrichtigungen'],
	'settings.tab.api': ['API', 'API', 'API', 'API'],
	'settings.tab.members': ['Members', 'Membres', 'Miembros', 'Mitglieder'],
	'settings.tab.advanced': ['Advanced', 'Avancé', 'Avanzado', 'Erweitert'],
	'profile.signedInVia': ['Signed in with', 'Connecté via', 'Sesión iniciada con', 'Angemeldet über'],
	'profile.facileAccount': ['Facile account', 'Compte Facile', 'Cuenta de Facile', 'Facile-Konto'],
	'profile.createdOn': ['Account created on', 'Compte créé le', 'Cuenta creada el', 'Konto erstellt am'],
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
	'profile.identity': ['Identity', 'Identité', 'Identidad', 'Identität'],
	'profile.identityText': [
		'The name and address other members of your calendars see.',
		'Le nom et l’adresse que voient les autres membres de vos calendriers.',
		'El nombre y la dirección que ven los demás miembros de tus calendarios.',
		'Der Name und die Adresse, die andere Mitglieder deiner Kalender sehen.'
	],
	'profile.nameText': [
		'Shown on the events you create and in member lists.',
		'Affiché sur les événements que vous créez et dans les listes de membres.',
		'Se muestra en los eventos que creas y en las listas de miembros.',
		'Wird bei Terminen, die du erstellst, und in Mitgliederlisten angezeigt.'
	],
	'profile.emailText': [
		'Your CalDAV username and the address people use to share a calendar with you.',
		'Votre identifiant CalDAV et l’adresse par laquelle on vous partage un calendrier.',
		'Tu usuario de CalDAV y la dirección con la que te comparten un calendario.',
		'Dein CalDAV-Benutzername und die Adresse, über die man einen Kalender mit dir teilt.'
	],
	'profile.photo': ['Profile picture', 'Photo de profil', 'Foto de perfil', 'Profilbild'],
	'profile.photoText': [
		'It comes from Registre and applies to the whole suite. Change it on your account; it shows here on the next load.',
		'Elle vient de Registre et vaut pour toute la suite. Changez-la sur votre compte, elle s’affiche ici au prochain chargement.',
		'Viene de Registre y vale para toda la suite. Cámbiala en tu cuenta; aparecerá aquí en la próxima carga.',
		'Es stammt aus Registre und gilt für die ganze Suite. Ändere es in deinem Konto; es erscheint hier beim nächsten Laden.'
	],
	'profile.photoEdit': ['Edit in Registre', 'Modifier sur Registre', 'Editar en Registre', 'In Registre ändern'],
	'profile.auth': ['Authentication', 'Authentification', 'Autenticación', 'Authentifizierung'],
	'profile.authText': [
		'How you get into Agenda. The server decides, not you.',
		'Comment vous entrez dans Agenda. C’est le serveur qui le décide, pas vous.',
		'Cómo entras en Agenda. Lo decide el servidor, no tú.',
		'Wie du in Agenda hineinkommst. Das entscheidet der Server, nicht du.'
	],
	'profile.authRow': [
		'Sign in with Facile',
		'Connexion avec Facile',
		'Inicio de sesión con Facile',
		'Anmeldung mit Facile'
	],
	'profile.authRowText': [
		'Agenda has no passwords. Your password, passkeys and security settings belong to your Facile account; change them in Registre.',
		'Agenda n’a pas de mot de passe. Votre mot de passe, vos passkeys et vos réglages de sécurité appartiennent à votre compte Facile : changez-les dans Registre.',
		'Agenda no tiene contraseñas. Tu contraseña, tus passkeys y tus ajustes de seguridad pertenecen a tu cuenta de Facile; cámbialos en Registre.',
		'Agenda kennt keine Passwörter. Passwort, Passkeys und Sicherheitseinstellungen gehören zu deinem Facile-Konto; ändere sie in Registre.'
	],
	'profile.authBadge': [
		'Managed by Registre',
		'Géré par Registre',
		'Gestionado por Registre',
		'Von Registre verwaltet'
	],
	'profile.session': ['Session', 'Session', 'Sesión', 'Sitzung'],
	'profile.sessionText': [
		'Sessions are per browser. Signing out here leaves your other devices alone.',
		'Les sessions sont par navigateur. Vous déconnecter ici laisse vos autres appareils tranquilles.',
		'Las sesiones son por navegador. Cerrar sesión aquí no afecta a tus otros dispositivos.',
		'Sitzungen gelten pro Browser. Die Abmeldung hier lässt deine anderen Geräte in Ruhe.'
	],
	'profile.logoutText': [
		'Ends this session and returns to the sign-in page. The API token keeps working; revoke it from the API tab.',
		'Termine cette session et renvoie vers la page de connexion. Le token API continue de fonctionner : révoquez-le depuis l’onglet API.',
		'Termina esta sesión y vuelve a la página de inicio de sesión. El token de API sigue funcionando; revócalo desde la pestaña API.',
		'Beendet diese Sitzung und kehrt zur Anmeldeseite zurück. Das API-Token funktioniert weiter; widerrufe es im Tab API.'
	],
	'profile.loggingOut': ['Signing out…', 'Déconnexion…', 'Cerrando sesión…', 'Abmelden…'],
	'appearance.title': ['Theme', 'Thème', 'Tema', 'Design'],
	'appearance.titleText': [
		'Stored in this browser. Each device you sign in on keeps its own choice.',
		'Stocké dans ce navigateur. Chaque appareil sur lequel vous vous connectez garde son propre choix.',
		'Se guarda en este navegador. Cada dispositivo en el que inicias sesión conserva su propia elección.',
		'In diesem Browser gespeichert. Jedes Gerät, auf dem du dich anmeldest, behält seine eigene Wahl.'
	],
	'appearance.scheme': ['Color scheme', 'Schéma de couleurs', 'Esquema de color', 'Farbschema'],
	'appearance.schemeText': [
		'System follows your operating system setting and keeps following it while the app is open.',
		'Système suit le réglage de votre système d’exploitation et continue de le suivre tant que l’app est ouverte.',
		'Sistema sigue el ajuste de tu sistema operativo y lo sigue mientras la app esté abierta.',
		'System folgt der Einstellung deines Betriebssystems und folgt ihr, solange die App geöffnet ist.'
	],
	'appearance.system': ['System', 'Système', 'Sistema', 'System'],
	'appearance.light': ['Light', 'Clair', 'Claro', 'Hell'],
	'appearance.dark': ['Dark', 'Sombre', 'Oscuro', 'Dunkel'],
	'appearance.current': ['Currently shown', 'Affiché actuellement', 'Mostrado ahora', 'Aktuell angezeigt'],
	'appearance.currentText': [
		'What the preference resolves to right now, once the system setting is taken into account.',
		'Ce que la préférence résout à cet instant, une fois le réglage système pris en compte.',
		'A qué se resuelve la preferencia en este momento, una vez tenido en cuenta el ajuste del sistema.',
		'Was die Einstellung gerade ergibt, nachdem die Systemeinstellung berücksichtigt wurde.'
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
