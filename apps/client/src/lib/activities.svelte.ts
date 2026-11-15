import { browser } from '$app/environment';
import { backend, type Activity } from '$lib/backend';

export type ActivitySource = Activity['source'];

const STORAGE_KEY = 'agenda:activity-overlays';
const SOURCES: ActivitySource[] = ['sablier', 'echo'];

function load(): Record<ActivitySource, boolean> {
	const shown = { sablier: true, echo: true };
	if (!browser) return shown;
	try {
		const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}') as Partial<Record<ActivitySource, boolean>>;
		for (const source of SOURCES) if (typeof saved[source] === 'boolean') shown[source] = saved[source];
	} catch {}
	return shown;
}

const overlays = $state(load());

/** Whether the strips of a source are drawn; remembered per browser. */
export function overlayShown(source: ActivitySource): boolean {
	return overlays[source];
}

/** Shows or hides the strips of a source. */
export function toggleOverlay(source: ActivitySource) {
	overlays[source] = !overlays[source];
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(overlays));
	} catch {}
}

/** The signed-in user's sessions and calls for the visible range, latest load wins. */
export class ActivityFeed {
	items = $state<Activity[]>([]);
	private seq = 0;

	/** Only the sources the person chose to see. */
	get visible(): Activity[] {
		return this.items.filter((a) => overlays[a.source]);
	}

	async load(range: { from: string; to: string }, wanted: boolean) {
		const seq = ++this.seq;
		if (!wanted || !SOURCES.some((s) => overlays[s])) {
			this.items = [];
			return;
		}
		try {
			const items = await backend.listActivities(range.from, range.to);
			if (seq === this.seq) this.items = items;
		} catch {
			if (seq === this.seq) this.items = [];
		}
	}
}
