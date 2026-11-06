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
	'api.token': ['API token', 'Token API', 'Token de API', 'API-Token'],
	'api.tokenText': [
		'A machine credential for CalDAV clients and HTTP calls. Agenda keeps one per account.',
		'Un identifiant machine pour les clients CalDAV et les appels HTTP. Agenda en garde un seul par compte.',
		'Una credencial de máquina para clientes CalDAV y llamadas HTTP. Agenda guarda una por cuenta.',
		'Eine Maschinenkennung für CalDAV-Clients und HTTP-Aufrufe. Agenda behält eines pro Konto.'
	],
	'api.tokenReplace': ['Replace the token', 'Remplacer le token', 'Reemplazar el token', 'Token ersetzen'],
	'api.tokenGenerate': ['Generate a token', 'Générer un token', 'Generar un token', 'Token erzeugen'],
	'api.tokenAlert': [
		'You sign in through Facile, so there is no Agenda password. Generate an API token and use it as the password in your CalDAV client.',
		'Vous vous connectez avec Facile : il n’y a pas de mot de passe Agenda. Générez un token API et utilisez-le comme mot de passe dans votre client CalDAV.',
		'Inicias sesión con Facile, así que no hay contraseña de Agenda. Genera un token de API y úsalo como contraseña en tu cliente CalDAV.',
		'Du meldest dich mit Facile an, daher gibt es kein Agenda-Passwort. Erzeuge ein API-Token und nutze es als Passwort in deinem CalDAV-Client.'
	],
	'api.colName': ['Name', 'Nom', 'Nombre', 'Name'],
	'api.colCreated': ['Created on', 'Créé le', 'Creado el', 'Erstellt am'],
	'api.colScope': ['Scope', 'Portée', 'Alcance', 'Geltungsbereich'],
	'api.colAction': ['Action', 'Action', 'Acción', 'Aktion'],
	'api.scope': [
		'CalDAV and API — your account',
		'CalDAV et API — votre compte',
		'CalDAV y API — tu cuenta',
		'CalDAV und API — dein Konto'
	],
	'api.noToken': ['No token', 'Aucun token', 'Ningún token', 'Kein Token'],
	'api.noTokenText': [
		'The token is shown once, when it is created. Agenda keeps only a hash and cannot show it to you again.',
		'Le token n’est affiché qu’une fois, à sa création. Agenda n’en stocke qu’un hash : il ne peut pas vous le rappeler ensuite.',
		'El token se muestra una sola vez, al crearlo. Agenda solo guarda un hash y no puede volver a mostrártelo.',
		'Das Token wird einmal angezeigt, bei der Erstellung. Agenda speichert nur einen Hash und kann es dir nicht erneut zeigen.'
	],
	'api.notGenerated': ['Not generated', 'Non généré', 'No generado', 'Nicht erzeugt'],
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
	'api.drawerReplaceTitle': [
		'Replace the API token',
		'Remplacer le token API',
		'Reemplazar el token de API',
		'API-Token ersetzen'
	],
	'api.drawerGenerateTitle': [
		'Generate an API token',
		'Générer un token API',
		'Generar un token de API',
		'API-Token erzeugen'
	],
	'api.drawerReplaceText': [
		'A new token immediately invalidates the old one. Name it after the machine that will use it.',
		'Générer un nouveau token invalidera immédiatement l’ancien. Nommez-le d’après la machine qui s’en servira.',
		'Un token nuevo invalida de inmediato el anterior. Ponle el nombre de la máquina que lo usará.',
		'Ein neues Token macht das alte sofort ungültig. Benenne es nach dem Rechner, der es nutzt.'
	],
	'api.drawerGenerateText': [
		'Name it after the machine that will use it; it is the only way to know later what you are revoking.',
		'Nommez-le d’après la machine qui s’en servira : c’est la seule façon de savoir plus tard ce que vous révoquez.',
		'Ponle el nombre de la máquina que lo usará: es la única forma de saber más tarde qué estás revocando.',
		'Benenne es nach dem Rechner, der es nutzt; nur so weißt du später, was du widerrufst.'
	],
	'api.copyNow': ['Copy it now', 'Copiez-le maintenant', 'Cópialo ahora', 'Jetzt kopieren'],
	'api.copyNowText': [
		'This token will never be shown again. Agenda keeps only a hash: once this drawer closes it cannot be recovered and you will have to generate another.',
		'Ce token ne sera plus jamais affiché. Agenda n’en conserve qu’un hash : une fois ce tiroir fermé, il est irrécupérable et il faudra en générer un autre.',
		'Este token no volverá a mostrarse. Agenda solo conserva un hash: al cerrar este panel no se puede recuperar y habrá que generar otro.',
		'Dieses Token wird nie wieder angezeigt. Agenda behält nur einen Hash: Nach dem Schließen dieses Fensters ist es verloren und du musst ein neues erzeugen.'
	],
	'api.tokenLabel': ['Token', 'Token', 'Token', 'Token'],
	'api.tokenHelper': [
		'Use it as the password in your CalDAV client.',
		'À utiliser comme mot de passe dans votre client CalDAV.',
		'Úsalo como contraseña en tu cliente CalDAV.',
		'Nutze es als Passwort in deinem CalDAV-Client.'
	],
	'api.tokenName': ['Token name', 'Nom du token', 'Nombre del token', 'Tokenname'],
	'api.tokenNamePlaceholder': [
		'MacBook, iPhone, CI server…',
		'MacBook, iPhone, serveur CI…',
		'MacBook, iPhone, servidor CI…',
		'MacBook, iPhone, CI-Server…'
	],
	'api.tokenScopeNote': [
		'Agenda attaches no scope or expiry to a token: it is worth your account, for CalDAV and for the API. Revoke it as soon as a machine no longer needs it.',
		'Agenda n’attache ni portée ni expiration à un token : il vaut votre compte, pour CalDAV comme pour l’API. Révoquez-le dès qu’une machine n’en a plus besoin.',
		'Agenda no asocia alcance ni caducidad a un token: equivale a tu cuenta, tanto para CalDAV como para la API. Revócalo en cuanto una máquina deje de necesitarlo.',
		'Agenda verknüpft kein Token mit einem Geltungsbereich oder Ablaufdatum: Es ist so mächtig wie dein Konto, für CalDAV wie für die API. Widerrufe es, sobald ein Rechner es nicht mehr braucht.'
	],
	'api.replaceButton': ['Replace', 'Remplacer', 'Reemplazar', 'Ersetzen'],
	'api.generateButton': ['Generate', 'Générer', 'Generar', 'Erzeugen'],
	'api.generating': ['Generating…', 'Génération…', 'Generando…', 'Wird erzeugt…'],
	'api.revokeTitle': [
		'Revoke the API token?',
		'Révoquer le token API ?',
		'¿Revocar el token de API?',
		'API-Token widerrufen?'
	],
	'api.revokeText': [
		'Every CalDAV client and script using it stops syncing immediately, and a revoked token cannot be restored: you will have to generate a new one and enter it again on each device.',
		'Tout client CalDAV et tout script qui s’en sert cesse de se synchroniser immédiatement, et un token révoqué ne se restaure pas : il faudra en générer un nouveau et le ressaisir sur chaque appareil.',
		'Todo cliente CalDAV y todo script que lo use deja de sincronizar de inmediato, y un token revocado no se puede restaurar: habrá que generar uno nuevo e introducirlo de nuevo en cada dispositivo.',
		'Jeder CalDAV-Client und jedes Skript, das es nutzt, hört sofort auf zu synchronisieren, und ein widerrufenes Token lässt sich nicht wiederherstellen: Du musst ein neues erzeugen und auf jedem Gerät neu eingeben.'
	]
} satisfies Record<string, Row>;
