import type { Row } from '../row';

/** The Ideas page: suggestions Bulle makes about the calendars, run in place. */
export const ideas = {
	'ideas.title': ['Ideas — Agenda', 'Idées — Agenda', 'Ideas — Agenda', 'Ideen — Agenda'],
	'ideas.description': [
		'Bulle looks at your calendars and suggests what to schedule, move or prepare. Run an idea and it is done here.',
		'Bulle regarde vos agendas et propose quoi planifier, déplacer ou préparer. Lancez une idée, elle se fait ici.',
		'Bulle revisa tus agendas y propone qué planificar, mover o preparar. Lanza una idea y se hace aquí.',
		'Bulle sieht sich Ihre Kalender an und schlägt vor, was zu planen, zu verschieben oder vorzubereiten ist. Starten Sie eine Idee, sie wird hier erledigt.'
	],
	'ideas.approval.summary': [
		'Bulle wants to run "{tool}". Check it before it goes ahead.',
		'Bulle veut lancer « {tool} ». Vérifiez avant de continuer.',
		'Bulle quiere ejecutar «{tool}». Revísalo antes de seguir.',
		'Bulle möchte „{tool}“ ausführen. Prüfen Sie es, bevor es weitergeht.'
	],
	'ideas.tool.agenda_create_event': ['Create an event', 'Créer un événement', 'Crear un evento', 'Termin anlegen'],
	'ideas.tool.agenda_update_event': ['Change an event', 'Modifier un événement', 'Modificar un evento', 'Termin ändern'],
	'ideas.tool.agenda_delete_event': ['Delete an event', 'Supprimer un événement', 'Eliminar un evento', 'Termin löschen'],
	'ideas.tool.agenda_read_event': ['Read an event', 'Lire un événement', 'Leer un evento', 'Termin lesen'],
	'ideas.tool.agenda_list_events': ['List events', 'Lister les événements', 'Listar eventos', 'Termine auflisten'],
	'ideas.tool.agenda_list_calendars': ['List calendars', 'Lister les agendas', 'Listar agendas', 'Kalender auflisten'],
	'ideas.tool.agenda_find_free_slots': ['Find free slots', 'Chercher des créneaux libres', 'Buscar huecos libres', 'Freie Zeiten suchen'],
	'ideas.error.load': [
		'Could not load the ideas.',
		'Impossible de charger les idées.',
		'No se pudieron cargar las ideas.',
		'Die Ideen konnten nicht geladen werden.'
	],
	'ideas.error.unreachable': [
		'Bulle is unreachable for now. Try again in a moment.',
		'Bulle est injoignable pour le moment. Réessayez dans un instant.',
		'Bulle no responde por ahora. Inténtalo en un momento.',
		'Bulle ist gerade nicht erreichbar. Versuchen Sie es gleich noch einmal.'
	],
	'ideas.error.notLinked': [
		'Reconnect your Facile account to use Bulle, or pick a space that Registre manages.',
		'Reconnectez votre compte Facile pour utiliser Bulle, ou choisissez un espace géré par Registre.',
		'Vuelve a conectar tu cuenta Facile para usar Bulle, o elige un espacio gestionado por Registre.',
		'Verbinden Sie Ihr Facile-Konto erneut oder wählen Sie einen von Registre verwalteten Bereich.'
	],
	'ideas.error.capped': [
		'You asked for ideas three times today. More come tomorrow.',
		'Vous avez demandé des idées trois fois aujourd’hui. D’autres arrivent demain.',
		'Ya pediste ideas tres veces hoy. Mañana llegan más.',
		'Sie haben heute dreimal nach Ideen gefragt. Morgen kommen neue.'
	],
	'ideas.error.busy': [
		'Bulle is already thinking of ideas for this space.',
		'Bulle cherche déjà des idées pour cet espace.',
		'Bulle ya está buscando ideas para este espacio.',
		'Bulle sucht bereits Ideen für diesen Bereich.'
	],
	'ideas.error.gone': [
		'This idea is no longer available.',
		'Cette idée n’est plus disponible.',
		'Esta idea ya no está disponible.',
		'Diese Idee ist nicht mehr verfügbar.'
	],
	'ideas.error.run': [
		'The run stopped before the end.',
		'L’exécution s’est arrêtée avant la fin.',
		'La ejecución se detuvo antes del final.',
		'Der Lauf wurde vor dem Ende abgebrochen.'
	],	'ideas.example.prep.title': [
		'Block time to prepare Thursday’s pitch',
		'Bloquer du temps pour préparer le pitch de jeudi',
		'Reservar tiempo para preparar el pitch del jueves',
		'Zeit für den Pitch am Donnerstag blocken'
	],
	'ideas.example.prep.why': [
		'The meeting is in three days and nothing is planned before it. Tomorrow morning is free.',
		'La réunion a lieu dans trois jours et rien n’est prévu avant. Demain matin est libre.',
		'La reunión es en tres días y no hay nada planificado antes. Mañana por la mañana está libre.',
		'Der Termin ist in drei Tagen und davor ist nichts geplant. Morgen Vormittag ist frei.'
	],
	'ideas.example.clash.title': [
		'Fix the double booking on Tuesday',
		'Résoudre le double rendez-vous de mardi',
		'Resolver la doble reserva del martes',
		'Doppelbuchung am Dienstag auflösen'
	],
	'ideas.example.clash.why': [
		'Two events overlap from 14:00 to 15:00. One of them could move to 16:00.',
		'Deux événements se chevauchent de 14 h à 15 h. L’un des deux peut passer à 16 h.',
		'Dos eventos se solapan de 14:00 a 15:00. Uno podría pasar a las 16:00.',
		'Zwei Termine überschneiden sich von 14 bis 15 Uhr. Einer könnte auf 16 Uhr rücken.'
	],
	'ideas.example.followup.title': [
		'Plan the follow-up call with Marc',
		'Planifier l’appel de suivi avec Marc',
		'Planificar la llamada de seguimiento con Marc',
		'Nachfassgespräch mit Marc planen'
	],
	'ideas.example.followup.why': [
		'Your last meeting with Marc was three weeks ago. Friday at 10:00 is open.',
		'Votre dernière réunion avec Marc date de trois semaines. Vendredi à 10 h est libre.',
		'Tu última reunión con Marc fue hace tres semanas. El viernes a las 10:00 está libre.',
		'Ihr letztes Treffen mit Marc war vor drei Wochen. Freitag um 10 Uhr ist frei.'
	],
	'ideas.dismissed': ['Idea dismissed', 'Idée écartée', 'Idea descartada', 'Idee verworfen']
} as const satisfies Record<string, Row>;
