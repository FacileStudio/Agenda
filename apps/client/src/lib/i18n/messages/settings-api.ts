import type { Row } from '../row';

/** Strings of the API settings tab. */
export const settingsApi = {
	'api.caldav': ['CalDAV connection', 'Connexion CalDAV', 'Conexión CalDAV', 'CalDAV-Verbindung'],
	'api.caldavText': [
		'Sync your calendars with iOS Calendar, macOS Calendar, DAVx⁵, Thunderbird or any other CalDAV client.',
		'Synchronisez vos calendriers avec Calendrier iOS, Calendrier macOS, DAVx⁵, Thunderbird ou tout autre client CalDAV.',
		'Sincroniza tus calendarios con Calendario de iOS, Calendario de macOS, DAVx⁵, Thunderbird o cualquier otro cliente CalDAV.',
		'Synchronisiere deine Kalender mit iOS Kalender, macOS Kalender, DAVx⁵, Thunderbird oder jedem anderen CalDAV-Client.'
	],
	'api.server': ['Server', 'Serveur', 'Servidor', 'Server'],
	'api.serverText': [
		'The host to enter when the client asks for an address rather than a full URL.',
		'L’hôte à saisir quand le client demande une adresse plutôt qu’une URL complète.',
		'El host que hay que introducir cuando el cliente pide una dirección en lugar de una URL completa.',
		'Der Host, den du eingibst, wenn der Client eine Adresse statt einer vollständigen URL verlangt.'
	],
	'api.path': ['Path', 'Chemin d’accès', 'Ruta', 'Pfad'],
	'api.pathText': [
		'Required by macOS Calendar in advanced CalDAV mode.',
		'Requis par Calendrier macOS en mode CalDAV avancé.',
		'Lo exige Calendario de macOS en el modo CalDAV avanzado.',
		'Wird von macOS Kalender im erweiterten CalDAV-Modus verlangt.'
	],
	'api.fullUrl': ['Full URL', 'URL complète', 'URL completa', 'Vollständige URL'],
	'api.fullUrlText': [
		'The base URL for DAVx⁵ and Thunderbird: server and path in one.',
		'L’URL de base pour DAVx⁵ et Thunderbird : serveur et chemin d’un seul tenant.',
		'La URL base para DAVx⁵ y Thunderbird: servidor y ruta en una sola.',
		'Die Basis-URL für DAVx⁵ und Thunderbird: Server und Pfad in einem.'
	],
	'api.username': ['Username', 'Identifiant', 'Usuario', 'Benutzername'],
	'api.usernameText': [
		'Your email address. CalDAV authentication is plain Basic Auth.',
		'Votre adresse email. L’authentification CalDAV est un Basic Auth classique.',
		'Tu dirección de correo. La autenticación CalDAV es un Basic Auth clásico.',
		'Deine E-Mail-Adresse. Die CalDAV-Authentifizierung ist klassisches Basic Auth.'
	],
	'api.password': ['Password', 'Mot de passe', 'Contraseña', 'Passwort'],
	'api.passwordText': [
		'You sign in through Facile, so Agenda has no password for you: the API token below stands in for it.',
		'Vous vous connectez avec Facile, Agenda n’a donc pas de mot de passe pour vous : le token API ci-dessous en tient lieu.',
		'Inicias sesión con Facile, así que Agenda no tiene contraseña para ti: el token de API de abajo la sustituye.',
		'Du meldest dich mit Facile an, daher hat Agenda kein Passwort für dich: das API-Token unten ersetzt es.'
	],
	'api.tokenRequired': ['API token required', 'Token API requis', 'Token de API necesario', 'API-Token erforderlich'],
	'api.colAction': ['Action', 'Action', 'Acción', 'Aktion'],
	'api.token': ['API token', 'Token API', 'Token de API', 'API-Token'],
	'api.tokenText': [
		'A machine credential for CalDAV clients and HTTP calls. Agenda keeps one per account.',
		'Un identifiant machine pour les clients CalDAV et les appels HTTP. Agenda en garde un seul par compte.',
		'Una credencial de máquina para clientes CalDAV y llamadas HTTP. Agenda guarda una por cuenta.',
		'Eine Maschinenkennung für CalDAV-Clients und HTTP-Aufrufe. Agenda behält eines pro Konto.'
	],
	'api.instructions': [
		'Instructions per client',
		'Instructions par client',
		'Instrucciones por cliente',
		'Anleitung je Client'
	],
	'api.instructionsText': [
		'The same credentials, in the order each app asks for them.',
		'Les mêmes identifiants, dans l’ordre où chaque application les demande.',
		'Las mismas credenciales, en el orden en que las pide cada aplicación.',
		'Dieselben Zugangsdaten, in der Reihenfolge, in der jede App sie verlangt.'
	],
	'api.client.ios': ['iOS — Calendar', 'iOS — Calendrier', 'iOS — Calendario', 'iOS — Kalender'],
	'api.client.macos': ['macOS — Calendar', 'macOS — Calendrier', 'macOS — Calendario', 'macOS — Kalender'],
	'api.client.davx5': ['Android — DAVx⁵', 'Android — DAVx⁵', 'Android — DAVx⁵', 'Android — DAVx⁵'],
	'api.client.thunderbird': ['Thunderbird', 'Thunderbird', 'Thunderbird', 'Thunderbird'],
	'api.step.ios1': [
		'Settings → Calendar → Accounts → Add Account → Other',
		'Réglages → Calendrier → Comptes → Ajouter un compte → Autre',
		'Ajustes → Calendario → Cuentas → Añadir cuenta → Otra',
		'Einstellungen → Kalender → Accounts → Account hinzufügen → Andere'
	],
	'api.step.ios2': ['Add a CalDAV account', 'Ajouter un compte CalDAV', 'Añadir una cuenta CalDAV', 'CalDAV-Account hinzufügen'],
	'api.step.server': ['Server: {value}', 'Serveur : {value}', 'Servidor: {value}', 'Server: {value}'],
	'api.step.user': [
		'Username: your email',
		'Nom d’utilisateur : votre email',
		'Usuario: tu correo',
		'Benutzername: deine E-Mail'
	],
	'api.step.password': [
		'Password: the API token above',
		'Mot de passe : le token API ci-dessus',
		'Contraseña: el token de API de arriba',
		'Passwort: das API-Token oben'
	],
	'api.step.macos1': [
		'Calendar → Settings → Accounts → + → CalDAV (Advanced)',
		'Calendrier → Réglages → Comptes → + → CalDAV (Avancé)',
		'Calendario → Ajustes → Cuentas → + → CalDAV (Avanzado)',
		'Kalender → Einstellungen → Accounts → + → CalDAV (Erweitert)'
	],
	'api.step.serverAddress': [
		'Server address: {value}',
		'Adresse du serveur : {value}',
		'Dirección del servidor: {value}',
		'Serveradresse: {value}'
	],
	'api.step.macosPath': ['Path: {value}', 'Chemin d’accès : {value}', 'Ruta: {value}', 'Pfad: {value}'],
	'api.step.macosPort': [
		'Port: 443 — SSL on',
		'Port : 443 — SSL activé',
		'Puerto: 443 — SSL activado',
		'Port: 443 — SSL aktiviert'
	],
	'api.step.davx51': [
		'DAVx⁵ → + → Login with URL and user name',
		'DAVx⁵ → + → Connexion avec URL et nom d’utilisateur',
		'DAVx⁵ → + → Iniciar sesión con URL y nombre de usuario',
		'DAVx⁵ → + → Anmeldung mit URL und Benutzername'
	],
	'api.step.baseUrl': ['Base URL: {value}', 'URL de base : {value}', 'URL base: {value}', 'Basis-URL: {value}'],
	'api.step.tb1': [
		'Calendar → New Calendar → On the Network → CalDAV',
		'Agenda → Nouveau calendrier → Sur le réseau → CalDAV',
		'Calendario → Nuevo calendario → En la red → CalDAV',
		'Kalender → Neuer Kalender → Im Netzwerk → CalDAV'
	],
	'api.step.location': ['Location: {value}', 'Emplacement : {value}', 'Ubicación: {value}', 'Adresse: {value}'],
	'api.step.tbUser': [
		'User name: your email',
		'Identifiant : votre email',
		'Nombre de usuario: tu correo',
		'Benutzername: deine E-Mail'
	],
	'api.toastReplaced': ['Token replaced.', 'Token remplacé.', 'Token reemplazado.', 'Token ersetzt.'],
	'api.toastGenerated': ['Token generated.', 'Token généré.', 'Token generado.', 'Token erzeugt.'],
	'api.toastRevoked': ['Token revoked.', 'Token révoqué.', 'Token revocado.', 'Token widerrufen.'],
} satisfies Record<string, Row>;
