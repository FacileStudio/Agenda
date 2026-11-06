import type { Row } from '../row';

/** Strings of the recurrence field and of the rule summaries. */
export const recurrence = {
	'rec.label': ['Repeat', 'Répétition', 'Repetición', 'Wiederholung'],
	'rec.none': ['Does not repeat', 'Ne se répète pas', 'No se repite', 'Wiederholt sich nicht'],
	'rec.custom': [
		'Custom (CalDAV client)',
		'Personnalisée (client CalDAV)',
		'Personalizada (cliente CalDAV)',
		'Benutzerdefiniert (CalDAV-Client)'
	],
	'rec.customSummary': [
		'Custom recurrence',
		'Récurrence personnalisée',
		'Recurrencia personalizada',
		'Benutzerdefinierte Wiederholung'
	],
	'rec.daily': ['Every day', 'Tous les jours', 'Todos los días', 'Jeden Tag'],
	'rec.weekly': ['Every week', 'Toutes les semaines', 'Todas las semanas', 'Jede Woche'],
	'rec.monthly': ['Every month', 'Tous les mois', 'Todos los meses', 'Jeden Monat'],
	'rec.yearly': ['Every year', 'Tous les ans', 'Todos los años', 'Jedes Jahr'],
	'rec.lead.daily': ['Every', 'Tous les', 'Cada', 'Alle'],
	'rec.lead.weekly': ['Every', 'Toutes les', 'Cada', 'Alle'],
	'rec.lead.monthly': ['Every', 'Tous les', 'Cada', 'Alle'],
	'rec.lead.yearly': ['Every', 'Tous les', 'Cada', 'Alle'],
	'rec.unit.daily': ['days', 'jours', 'días', 'Tage'],
	'rec.unit.weekly': ['weeks', 'semaines', 'semanas', 'Wochen'],
	'rec.unit.monthly': ['months', 'mois', 'meses', 'Monate'],
	'rec.unit.yearly': ['years', 'ans', 'años', 'Jahre'],
	'rec.interval': ['Interval', 'Intervalle', 'Intervalo', 'Intervall'],
	'rec.weekdays': ['Days of the week', 'Jours de la semaine', 'Días de la semana', 'Wochentage'],
	'rec.ends': ['Ends', 'Se termine', 'Termina', 'Endet'],
	'rec.endNever': ['never', 'jamais', 'nunca', 'nie'],
	'rec.endUntil': ['on', 'le', 'el', 'am'],
	'rec.endCount': ['after', 'après', 'después de', 'nach'],
	'rec.endDate': ['End date', 'Date de fin', 'Fecha de fin', 'Enddatum'],
	'rec.occurrencesAria': [
		'Number of occurrences',
		'Nombre d’occurrences',
		'Número de repeticiones',
		'Anzahl der Wiederholungen'
	],
	'rec.times': ['times', 'fois', 'veces', 'Mal'],
	'rec.workdays': ['Monday to Friday', 'Du lundi au vendredi', 'De lunes a viernes', 'Montag bis Freitag'],
	'rec.onDays': [' on {days}', ' le {days}', ' el {days}', ' am {days}'],
	'rec.count': [
		{ one: ', {count} time', other: ', {count} times' },
		{ one: ', {count} fois', other: ', {count} fois' },
		{ one: ', {count} vez', other: ', {count} veces' },
		{ one: ', {count} Mal', other: ', {count} Mal' }
	],
	'rec.until': [', until {date}', ', jusqu’au {date}', ', hasta el {date}', ', bis zum {date}']
} satisfies Record<string, Row>;
