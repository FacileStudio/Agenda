import type { Row } from '../row';

/** Strings of the Members and Advanced settings tabs. */
export const settingsAdmin = {
	'members.calendars': ['Calendars', 'Calendriers', 'Calendarios', 'Kalender'],
	'members.calendarsText': [
		'A personal calendar is shared by email address. A space’s calendar is shared with all its members.',
		'Un calendrier personnel se partage par adresse email. Celui d’un espace est partagé avec tous ses membres.',
		'Un calendario personal se comparte por dirección de correo. El de un espacio se comparte con todos sus miembros.',
		'Ein persönlicher Kalender wird per E-Mail-Adresse geteilt. Der Kalender eines Bereichs wird mit allen Mitgliedern geteilt.'
	],
	'members.newCalendar': ['New calendar', 'Nouveau calendrier', 'Nuevo calendario', 'Neuer Kalender'],
	'members.empty': [
		'No calendar yet. Create one to start sharing.',
		'Aucun calendrier pour l’instant. Créez-en un pour commencer à partager.',
		'Todavía no hay calendarios. Crea uno para empezar a compartir.',
		'Noch kein Kalender. Erstelle einen, um mit dem Teilen zu beginnen.'
	],
	'members.colCalendar': ['Calendar', 'Calendrier', 'Calendario', 'Kalender'],
	'members.colType': ['Type', 'Type', 'Tipo', 'Typ'],
	'members.colRole': ['Your role', 'Votre rôle', 'Tu rol', 'Deine Rolle'],
	'members.typeSpace': ['Space', 'Espace', 'Espacio', 'Bereich'],
	'members.typePersonal': ['Personal', 'Personnel', 'Personal', 'Persönlich'],
	'members.typeShared': ['Shared', 'Partagé', 'Compartido', 'Geteilt'],
	'members.spaceMembers': [
		'Space members',
		'Membres de l’espace',
		'Miembros del espacio',
		'Bereichsmitglieder'
	],
	'members.manage': ['Manage', 'Gérer', 'Gestionar', 'Verwalten'],
	'members.manageAria': ['Manage {name}', 'Gérer {name}', 'Gestionar {name}', '{name} verwalten'],
	'members.spaces': ['Spaces', 'Espaces', 'Espacios', 'Bereiche'],
	'members.spacesText': [
		'Each space has its own calendar, shared with all its members. It is born with the space and carries its name.',
		'Chaque espace a son calendrier, partagé avec tous ses membres. Il naît avec l’espace et en porte le nom.',
		'Cada espacio tiene su calendario, compartido con todos sus miembros. Nace con el espacio y lleva su nombre.',
		'Jeder Bereich hat einen eigenen Kalender, der mit allen Mitgliedern geteilt wird. Er entsteht mit dem Bereich und trägt dessen Namen.'
	],
	'members.allSpaces': ['All spaces', 'Tous les espaces', 'Todos los espacios', 'Alle Bereiche'],
	'members.spaceRow': ['Members of a space', 'Membres d’un espace', 'Miembros de un espacio', 'Mitglieder eines Bereichs'],
	'members.spaceRowText': [
		'Inviting, changing a role or removing someone happens in Registre, for every Facile app at once. The space’s calendar follows.',
		'Inviter, changer un rôle ou retirer quelqu’un se fait dans Registre, pour toutes les apps Facile à la fois. Le calendrier de l’espace suit.',
		'Invitar, cambiar un rol o quitar a alguien se hace en Registre, para todas las apps de Facile a la vez. El calendario del espacio lo sigue.',
		'Einladen, eine Rolle ändern oder jemanden entfernen geschieht in Registre, für alle Facile-Apps zugleich. Der Kalender des Bereichs folgt.'
	],
	'members.seeSpaces': ['See spaces', 'Voir les espaces', 'Ver espacios', 'Bereiche ansehen'],
	'members.toastCreated': ['Calendar created.', 'Calendrier créé.', 'Calendario creado.', 'Kalender erstellt.'],
	'members.toastUpdated': [
		'Calendar updated.',
		'Calendrier mis à jour.',
		'Calendario actualizado.',
		'Kalender aktualisiert.'
	],
	'members.toastDeleted': [
		'Calendar deleted.',
		'Calendrier supprimé.',
		'Calendario eliminado.',
		'Kalender gelöscht.'
	],
	'advanced.instance': ['Instance', 'Instance', 'Instancia', 'Instanz'],
	'advanced.instanceText': [
		'The facts to quote when you open a ticket about a self-hosted install.',
		'Les faits à citer quand vous ouvrez un ticket sur une installation auto-hébergée.',
		'Los datos que citar al abrir un ticket sobre una instalación autoalojada.',
		'Die Angaben, die du nennst, wenn du ein Ticket zu einer selbst gehosteten Installation öffnest.'
	],
	'advanced.interface': ['Interface', 'Interface', 'Interfaz', 'Oberfläche'],
	'advanced.interfaceText': [
		'This application is the web client of the Agenda API, served by the same binary.',
		'Cette application est le client web de l’API Agenda, servi par le même binaire.',
		'Esta aplicación es el cliente web de la API de Agenda, servido por el mismo binario.',
		'Diese Anwendung ist der Web-Client der Agenda-API, ausgeliefert von derselben Binärdatei.'
	],
	'advanced.sso': ['Single sign-on', 'Authentification unique', 'Inicio de sesión único', 'Single Sign-on'],
	'advanced.ssoText': [
		'OIDC federation to Registre, the only way to sign in. Enabled by OIDC_ISSUER on the server.',
		'Fédération OIDC vers Registre, la seule façon de se connecter. Activée par OIDC_ISSUER côté serveur.',
		'Federación OIDC con Registre, la única forma de iniciar sesión. Se activa con OIDC_ISSUER en el servidor.',
		'OIDC-Föderation zu Registre, die einzige Anmeldemöglichkeit. Wird serverseitig durch OIDC_ISSUER aktiviert.'
	],
	'advanced.enabled': ['Enabled', 'Activée', 'Activado', 'Aktiviert'],
	'advanced.notConfigured': ['Not configured', 'Non configurée', 'Sin configurar', 'Nicht konfiguriert'],
	'advanced.checking': ['Checking…', 'Vérification…', 'Comprobando…', 'Prüfe…'],
	'advanced.encryption': [
		'OIDC token encryption',
		'Chiffrement des jetons OIDC',
		'Cifrado de los tokens OIDC',
		'Verschlüsselung der OIDC-Tokens'
	],
	'advanced.encryptionText': [
		'ENCRYPTION_KEY encrypts the identity provider’s tokens at rest. Without it they are stored in clear.',
		'ENCRYPTION_KEY chiffre au repos les jetons rendus par le fournisseur d’identité. Sans elle ils sont stockés en clair.',
		'ENCRYPTION_KEY cifra en reposo los tokens del proveedor de identidad. Sin ella se guardan en claro.',
		'ENCRYPTION_KEY verschlüsselt die Tokens des Identitätsanbieters im Ruhezustand. Ohne sie werden sie im Klartext gespeichert.'
	],
	'advanced.keySet': ['Key set', 'Clé définie', 'Clave definida', 'Schlüssel gesetzt'],
	'advanced.keyMissing': ['No key', 'Aucune clé', 'Sin clave', 'Kein Schlüssel'],
	'advanced.interop': ['Interoperability', 'Interopérabilité', 'Interoperabilidad', 'Interoperabilität'],
	'advanced.interopText': [
		'What other software can read without going through this interface.',
		'Ce que d’autres logiciels peuvent lire sans passer par cette interface.',
		'Lo que otro software puede leer sin pasar por esta interfaz.',
		'Was andere Software lesen kann, ohne diese Oberfläche zu nutzen.'
	],
	'advanced.endpoint': ['CalDAV endpoint', 'Point d’accès CalDAV', 'Punto de acceso CalDAV', 'CalDAV-Endpunkt'],
	'advanced.endpointText': [
		'The root of the CalDAV server. Credentials and per-client details are in the API tab.',
		'Racine du serveur CalDAV. Les identifiants et le détail par client sont dans l’onglet API.',
		'La raíz del servidor CalDAV. Las credenciales y el detalle por cliente están en la pestaña API.',
		'Die Wurzel des CalDAV-Servers. Zugangsdaten und Details je Client stehen im Tab API.'
	],
	'advanced.discovery': ['Discovery', 'Découverte', 'Descubrimiento', 'Erkennung'],
	'advanced.discoveryText': [
		'Clients that can do it find the server on their own from the domain.',
		'Les clients qui savent le faire trouvent le serveur tout seuls à partir du domaine.',
		'Los clientes que saben hacerlo encuentran el servidor por sí solos a partir del dominio.',
		'Clients, die das können, finden den Server allein anhand der Domain.'
	],
	'advanced.export': ['Calendar export', 'Export d’un calendrier', 'Exportar un calendario', 'Kalenderexport'],
	'advanced.exportText': [
		'A CalDAV client fetches the original ICS, alarms and recurrences included; Agenda has no separate export to maintain.',
		'Un client CalDAV récupère l’ICS d’origine, alarmes et récurrences comprises : Agenda n’a pas d’export séparé à maintenir.',
		'Un cliente CalDAV obtiene el ICS original, con alarmas y recurrencias incluidas: Agenda no tiene una exportación aparte que mantener.',
		'Ein CalDAV-Client holt das Original-ICS samt Alarmen und Wiederholungen; Agenda muss keinen separaten Export pflegen.'
	],
	'advanced.connect': ['Connect a client', 'Connecter un client', 'Conectar un cliente', 'Client verbinden'],
	'advanced.danger': ['Danger zone', 'Zone de danger', 'Zona de peligro', 'Gefahrenzone'],
	'advanced.dangerText': [
		'Irreversible actions live at the bottom of this tab, never in the tab bar.',
		'Les actions irréversibles vivent en bas de cet onglet — jamais dans la barre d’onglets.',
		'Las acciones irreversibles viven al final de esta pestaña, nunca en la barra de pestañas.',
		'Unumkehrbare Aktionen stehen am Ende dieses Tabs, nie in der Tab-Leiste.'
	],
	'advanced.dangerAlert': [
		'Agenda has nothing destructive at account level: an account is an identity and owns no data of its own. What destroys belongs to the object concerned (a calendar, a space, a token) and happens where that object is managed.',
		'Agenda n’a rien de destructif au niveau du compte : un compte est une identité, il ne possède aucune donnée en propre. Ce qui détruit appartient à l’objet concerné — un calendrier, un espace, un token — et se fait là où cet objet se gère.',
		'Agenda no tiene nada destructivo a nivel de cuenta: una cuenta es una identidad y no posee datos propios. Lo que destruye pertenece al objeto afectado (un calendario, un espacio, un token) y se hace donde se gestiona ese objeto.',
		'Agenda hat auf Kontoebene nichts Zerstörerisches: Ein Konto ist eine Identität und besitzt keine eigenen Daten. Was zerstört, gehört zum jeweiligen Objekt (Kalender, Bereich, Token) und geschieht dort, wo dieses Objekt verwaltet wird.'
	],
	'advanced.deleteCalendar': [
		'Delete a calendar',
		'Supprimer un calendrier',
		'Eliminar un calendario',
		'Kalender löschen'
	],
	'advanced.deleteCalendarText': [
		'Takes its events with it and unsubscribes every CalDAV client that synced it. Owner only, from the calendar’s card.',
		'Emporte ses événements et désabonne tous les clients CalDAV qui le synchronisaient. Réservé au propriétaire, depuis la fiche du calendrier.',
		'Se lleva sus eventos y da de baja a todos los clientes CalDAV que lo sincronizaban. Solo el propietario, desde la ficha del calendario.',
		'Nimmt die Termine mit und meldet alle CalDAV-Clients ab, die ihn synchronisiert haben. Nur für den Eigentümer, über die Kalenderkarte.'
	],
	'advanced.openCalendars': ['Open calendars', 'Ouvrir les calendriers', 'Abrir calendarios', 'Kalender öffnen'],
	'advanced.revokeToken': [
		'Revoke the API token',
		'Révoquer le token API',
		'Revocar el token de API',
		'API-Token widerrufen'
	],
	'advanced.revokeTokenText': [
		'Every client or script using it stops syncing immediately, and a revoked token cannot be restored.',
		'Tout client ou script qui s’en sert cesse de se synchroniser immédiatement, et un token révoqué ne se restaure pas.',
		'Todo cliente o script que lo use deja de sincronizar de inmediato, y un token revocado no se puede restaurar.',
		'Jeder Client oder jedes Skript, das es nutzt, hört sofort auf zu synchronisieren, und ein widerrufenes Token lässt sich nicht wiederherstellen.'
	],
	'advanced.openApi': ['Open the API tab', 'Ouvrir l’onglet API', 'Abrir la pestaña API', 'Tab API öffnen'],
	'advanced.leaveSpace': ['Leave a space', 'Quitter un espace', 'Salir de un espacio', 'Bereich verlassen'],
	'advanced.leaveSpaceText': [
		'Leaving a space cuts you off from its calendar in every Facile app. It happens in Registre.',
		'Quitter un espace vous coupe de son calendrier, dans toutes les apps Facile. Cela se fait dans Registre.',
		'Salir de un espacio te desconecta de su calendario en todas las apps de Facile. Se hace en Registre.',
		'Wenn du einen Bereich verlässt, verlierst du den Zugriff auf seinen Kalender in allen Facile-Apps. Das geschieht in Registre.'
	],
	'advanced.deleteAccount': [
		'Delete the account',
		'Supprimer le compte',
		'Eliminar la cuenta',
		'Konto löschen'
	],
	'advanced.deleteAccountText': [
		'Agenda has no endpoint for it: deleting an account happens in your Facile account, in Registre.',
		'Agenda n’a pas de point d’accès pour cela : la suppression d’un compte se fait sur votre compte Facile, dans Registre.',
		'Agenda no tiene un punto de acceso para ello: la eliminación de una cuenta se hace en tu cuenta de Facile, en Registre.',
		'Agenda hat dafür keinen Endpunkt: Ein Konto wird in deinem Facile-Konto in Registre gelöscht.'
	],
	'advanced.viaRegistre': ['Through Registre', 'Via Registre', 'A través de Registre', 'Über Registre']
} satisfies Record<string, Row>;
