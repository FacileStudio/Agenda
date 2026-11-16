import type { Row } from '../row';

/** The Bulle button of the calendar: planning runs, the draft plan and its review. */
export const bulle = {
	'bulle.title': ['Plan with {name}', 'Planifier avec {name}', 'Planificar con {name}', 'Planen mit {name}'],
	'bulle.close': ['Close', 'Fermer', 'Cerrar', 'Schließen'],
	'bulle.intro': [
		'{name} reads your Opus tasks and your free time, then proposes a plan. Nothing reaches your calendar until you accept it.',
		'{name} lit vos tâches Opus et votre temps libre, puis propose un plan. Rien n’arrive dans votre agenda avant que vous l’acceptiez.',
		'{name} lee tus tareas de Opus y tu tiempo libre, y propone un plan. Nada llega a tu agenda hasta que lo aceptes.',
		'{name} liest Ihre Opus-Aufgaben und Ihre freie Zeit und schlägt einen Plan vor. Nichts landet in Ihrem Kalender, bevor Sie zustimmen.'
	],
	'bulle.action.week': ['Plan my week', 'Planifier ma semaine', 'Planificar mi semana', 'Meine Woche planen'],
	'bulle.action.today': ['Plan today', 'Planifier aujourd’hui', 'Planificar hoy', 'Heute planen'],
	'bulle.ask.label': ['Or ask for something else', 'Ou demandez autre chose', 'O pide otra cosa', 'Oder bitten Sie um etwas anderes'],
	'bulle.ask.placeholder': [
		'For example: put the report on Thursday morning',
		'Par exemple : mets le rapport jeudi matin',
		'Por ejemplo: pon el informe el jueves por la mañana',
		'Zum Beispiel: den Bericht auf Donnerstagvormittag legen'
	],
	'bulle.ask.send': ['Send', 'Envoyer', 'Enviar', 'Senden'],
	'bulle.run.done': [
		'Done. The draft is below and in your calendar as pale blocks.',
		'Terminé. Le brouillon est ci-dessous et dans votre agenda en blocs pâles.',
		'Listo. El borrador está debajo y en tu agenda como bloques pálidos.',
		'Fertig. Der Entwurf steht unten und als blasse Blöcke in Ihrem Kalender.'
	],
	'bulle.run.noPlan': [
		'{name} finished without saving a plan. Try again, or ask for something more precise.',
		'{name} a terminé sans enregistrer de plan. Réessayez, ou demandez quelque chose de plus précis.',
		'{name} terminó sin guardar un plan. Inténtalo de nuevo o pide algo más preciso.',
		'{name} wurde fertig, ohne einen Plan zu speichern. Versuchen Sie es erneut oder fragen Sie genauer.'
	],
	'bulle.run.failed': [
		'The planning run failed.',
		'La planification a échoué.',
		'La planificación falló.',
		'Die Planung ist fehlgeschlagen.'
	],
	'bulle.run.continue': ['Continue in Bulle', 'Continuer dans Bulle', 'Continuar en Bulle', 'In Bulle fortfahren'],
	'bulle.error.start': [
		'Could not start the run. Try again.',
		'Impossible de lancer la planification. Réessayez.',
		'No se pudo iniciar la planificación. Inténtalo de nuevo.',
		'Die Planung konnte nicht gestartet werden. Versuchen Sie es erneut.'
	],
	'bulle.error.plan': [
		'Could not load the draft plan.',
		'Impossible de charger le brouillon.',
		'No se pudo cargar el borrador.',
		'Der Entwurf konnte nicht geladen werden.'
	],
	'bulle.error.apply': [
		'Could not add the blocks to your calendar.',
		'Impossible d’ajouter les blocs à votre agenda.',
		'No se pudieron añadir los bloques a tu agenda.',
		'Die Blöcke konnten nicht in Ihren Kalender übernommen werden.'
	],
	'bulle.error.expired': [
		'This plan is no longer available. Ask for a new one.',
		'Ce plan n’est plus disponible. Demandez-en un nouveau.',
		'Este plan ya no está disponible. Pide uno nuevo.',
		'Dieser Plan ist nicht mehr verfügbar. Fordern Sie einen neuen an.'
	],
	'bulle.upsell': [
		'Planning with {name} needs a plan with AI. Your calendar and your tasks keep working without it.',
		'Planifier avec {name} demande une offre avec IA. Votre agenda et vos tâches fonctionnent sans.',
		'Planificar con {name} requiere un plan con IA. Tu agenda y tus tareas siguen funcionando sin él.',
		'Planen mit {name} braucht einen Tarif mit KI. Kalender und Aufgaben funktionieren auch ohne.'
	],
	'bulle.plan.heading': ['Draft plan', 'Brouillon de plan', 'Borrador del plan', 'Planentwurf'],
	'bulle.plan.summary': [
		'{count} blocks proposed',
		'{count} blocs proposés',
		'{count} bloques propuestos',
		'{count} Blöcke vorgeschlagen'
	],
	'bulle.plan.bar': [
		'{name} proposed {count} blocks. Nothing is in your calendar yet.',
		'{name} propose {count} blocs. Rien n’est encore dans votre agenda.',
		'{name} propone {count} bloques. Todavía no hay nada en tu agenda.',
		'{name} schlägt {count} Blöcke vor. Noch ist nichts in Ihrem Kalender.'
	],
	'bulle.plan.review': ['Review', 'Revoir', 'Revisar', 'Prüfen'],
	'bulle.plan.acceptAll': ['Accept all', 'Tout accepter', 'Aceptar todo', 'Alle übernehmen'],
	'bulle.plan.accept': ['Accept {count}', 'Accepter {count}', 'Aceptar {count}', '{count} übernehmen'],
	'bulle.plan.dismiss': ['Dismiss', 'Écarter', 'Descartar', 'Verwerfen'],
	'bulle.plan.keep': ['Keep', 'Garder', 'Mantener', 'Behalten'],
	'bulle.plan.drop': ['Drop', 'Retirer', 'Quitar', 'Entfernen'],
	'bulle.plan.dropped': ['Dropped', 'Retiré', 'Quitado', 'Entfernt'],
	'bulle.plan.warnings': ['Warnings', 'Avertissements', 'Avisos', 'Hinweise'],
	'bulle.plan.overlap': ['Overlaps an event', 'Chevauche un événement', 'Se solapa con un evento', 'Überschneidet einen Termin'],
	'bulle.plan.conflict': [
		'These blocks now overlap an event: {titles}.',
		'Ces blocs chevauchent maintenant un événement : {titles}.',
		'Estos bloques ahora se solapan con un evento: {titles}.',
		'Diese Blöcke überschneiden jetzt einen Termin: {titles}.'
	],
	'bulle.plan.conflictGeneric': [
		'Some blocks now overlap an event.',
		'Certains blocs chevauchent maintenant un événement.',
		'Algunos bloques ahora se solapan con un evento.',
		'Einige Blöcke überschneiden jetzt einen Termin.'
	],
	'bulle.plan.force': ['Apply anyway', 'Appliquer quand même', 'Aplicar igualmente', 'Trotzdem übernehmen'],
	'bulle.plan.leaveOut': ['Leave them out', 'Les laisser de côté', 'Dejarlos fuera', 'Weglassen'],
	'bulle.plan.applied': [
		'Added {count} events to your calendar.',
		'{count} événements ajoutés à votre agenda.',
		'{count} eventos añadidos a tu agenda.',
		'{count} Termine in Ihren Kalender übernommen.'
	],
	'bulle.plan.dismissed': ['Plan dismissed.', 'Plan écarté.', 'Plan descartado.', 'Plan verworfen.'],
	'bulle.plan.reason': ['Why: {reason}', 'Pourquoi : {reason}', 'Por qué: {reason}', 'Warum: {reason}'],
	'bulle.ghost.aria': [
		'Proposed: {title}, {time}. Open the plan.',
		'Proposé : {title}, {time}. Ouvrir le plan.',
		'Propuesto: {title}, {time}. Abrir el plan.',
		'Vorgeschlagen: {title}, {time}. Plan öffnen.'
	],
	'bulle.approvals': ['Waiting for you', 'En attente de vous', 'Esperando tu respuesta', 'Wartet auf Sie'],
	'ideas.tool.agenda_propose_plan': ['Save a plan draft', 'Enregistrer un brouillon de plan', 'Guardar un borrador de plan', 'Planentwurf speichern'],
	'ideas.tool.agenda_list_plans': ['List plan drafts', 'Lister les brouillons de plan', 'Listar borradores de plan', 'Planentwürfe auflisten'],
	'ideas.tool.opus_search_tasks': ['Search Opus tasks', 'Chercher dans les tâches Opus', 'Buscar tareas de Opus', 'Opus-Aufgaben suchen']
} as const satisfies Record<string, Row>;
