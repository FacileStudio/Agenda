let busy: () => boolean = () => false;
const waiting = new Set<() => void>();

/**
 * Declares what makes the view busy (a drag, an open dialog). While it holds, live reloads
 * wait, so a change never yanks an event from under the pointer. Returns the undo.
 */
export function setViewBusy(check: () => boolean): () => void {
	busy = check;
	return () => {
		busy = () => false;
		settleReloads();
	};
}

/** Runs reload now, or once the view stops being busy. Repeated requests run it once. */
export function requestReload(reload: () => void) {
	if (busy()) waiting.add(reload);
	else reload();
}

/** Runs the reloads that waited, if the view is no longer busy. */
export function settleReloads() {
	if (busy() || waiting.size === 0) return;
	const due = [...waiting];
	waiting.clear();
	due.forEach((reload) => reload());
}
