/** One merged notice from GET /api/live: what changed during the debounce window. */
export type LiveChange = { calendars: boolean; events: boolean; activities: boolean };

type Frame = { calendar_id?: number; calendar?: boolean; activity?: boolean; resync?: boolean };

const DEBOUNCE_MS = 300;
const RETRY_MS = [1000, 3000, 10000, 30000];

const listeners = new Set<(change: LiveChange) => void>();
let pending: LiveChange | null = null;
let timer: ReturnType<typeof setTimeout> | undefined;

/** Calls fn with each debounced change; returns the unsubscribe. */
export function onLiveChange(fn: (change: LiveChange) => void): () => void {
	listeners.add(fn);
	return () => listeners.delete(fn);
}

function queue(frame: Frame) {
	const everything = frame.resync === true;
	pending ??= { calendars: false, events: false, activities: false };
	pending.calendars ||= everything || frame.calendar === true;
	pending.events ||= everything || frame.calendar_id !== undefined;
	pending.activities ||= everything || frame.activity === true;
	clearTimeout(timer);
	timer = setTimeout(() => {
		const change = pending;
		pending = null;
		if (change) listeners.forEach((fn) => fn(change));
	}, DEBOUNCE_MS);
}

/**
 * Opens the stream and keeps it open until the returned stop is called. The browser
 * reconnects a dropped stream by itself, but gives up for good on an HTTP error such as
 * the 502 a deploy answers, so a closed stream is reopened here with a growing delay.
 * Coming back to the tab or back online counts as a resync, since the stream may have
 * missed changes meanwhile.
 */
export function startLive(url = '/api/live'): () => void {
	let source: EventSource | null = null;
	let attempt = 0;
	let retry: ReturnType<typeof setTimeout> | undefined;
	let stopped = false;
	const open = () => {
		source = new EventSource(url, { withCredentials: true });
		source.onopen = () => (attempt = 0);
		source.addEventListener('change', (e) => queue(parseFrame((e as MessageEvent).data)));
		source.onerror = () => {
			if (stopped || source?.readyState !== EventSource.CLOSED) return;
			retry = setTimeout(open, RETRY_MS[Math.min(attempt++, RETRY_MS.length - 1)]);
		};
	};
	const resync = () => {
		if (document.visibilityState === 'visible') queue({ resync: true });
	};
	open();
	document.addEventListener('visibilitychange', resync);
	window.addEventListener('online', resync);
	return () => {
		stopped = true;
		clearTimeout(retry);
		source?.close();
		document.removeEventListener('visibilitychange', resync);
		window.removeEventListener('online', resync);
	};
}

function parseFrame(data: string): Frame {
	try {
		return JSON.parse(data) as Frame;
	} catch {
		return { resync: true };
	}
}
