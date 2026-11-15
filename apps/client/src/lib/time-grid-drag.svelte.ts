import type { AgendaEvent } from '$lib/backend';
import {
	DRAG_THRESHOLD_PX,
	SNAP_MINUTES,
	atMinutes,
	drawRange,
	minutesOfDay,
	moveRange,
	resizeRange,
	type MinuteRange
} from '$lib/week-drag';

type Mode = 'create' | 'move' | 'resize-top' | 'resize-bottom';

type Gesture = {
	mode: Mode;
	column: number;
	startX: number;
	startY: number;
	anchor: number;
	origin: MinuteRange;
	event: AgendaEvent | null;
	active: boolean;
};

type Options = {
	columns: () => (HTMLElement | undefined)[];
	days: () => Date[];
	slotHeight: number;
	canEdit: (event: AgendaEvent) => boolean;
	onCreate: (start: Date, end: Date) => void;
	onChange: (event: AgendaEvent, start: Date, end: Date) => void;
};

export class TimeGridDrag {
	gesture = $state<Gesture | null>(null);
	preview = $state<{ column: number; range: MinuteRange } | null>(null);
	private options: Options;

	constructor(options: Options) {
		this.options = options;
	}

	get active(): boolean {
		return this.gesture?.active === true;
	}

	isDragging(event: AgendaEvent): boolean {
		return this.gesture?.active === true && this.gesture.event === event;
	}

	startCreate = (e: PointerEvent, column: number) => {
		if (!this.pointerAllowed(e)) return;
		const minute = this.minuteAt(e.clientY, column);
		this.begin(e, {
			mode: 'create',
			column,
			startX: e.clientX,
			startY: e.clientY,
			anchor: minute,
			origin: { start: minute, end: minute },
			event: null,
			active: false
		});
	};

	startEvent = (e: PointerEvent, event: AgendaEvent, mode: Exclude<Mode, 'create'>, column: number) => {
		if (!this.options.canEdit(event) || !this.pointerAllowed(e)) return;
		e.stopPropagation();
		const start = new Date(event.start_at);
		const startMin = minutesOfDay(start);
		const endMin = Math.max(minutesOfDay(new Date(event.end_at)), startMin + SNAP_MINUTES);
		this.begin(e, {
			mode,
			column,
			startX: e.clientX,
			startY: e.clientY,
			anchor: 0,
			origin: { start: startMin, end: endMin },
			event,
			active: false
		});
	};

	private pointerAllowed(e: PointerEvent): boolean {
		return e.button === 0 && e.pointerType !== 'touch';
	}

	private begin(e: PointerEvent, gesture: Gesture) {
		e.preventDefault();
		this.gesture = gesture;
		window.addEventListener('pointermove', this.onMove);
		window.addEventListener('pointerup', this.onUp);
		window.addEventListener('pointercancel', this.cancel);
		window.addEventListener('keydown', this.onKey);
	}

	private finish() {
		window.removeEventListener('pointermove', this.onMove);
		window.removeEventListener('pointerup', this.onUp);
		window.removeEventListener('pointercancel', this.cancel);
		window.removeEventListener('keydown', this.onKey);
		this.gesture = null;
		this.preview = null;
	}

	private swallowClick() {
		const stop = (e: Event) => {
			e.stopPropagation();
			e.preventDefault();
		};
		window.addEventListener('click', stop, { capture: true, once: true });
		setTimeout(() => window.removeEventListener('click', stop, { capture: true }), 0);
	}

	minuteAt(clientY: number, column: number): number {
		const el = this.options.columns()[column];
		if (!el) return 0;
		const top = el.getBoundingClientRect().top;
		const minutes = ((clientY - top) / this.options.slotHeight) * 60;
		return Math.min(Math.max(minutes, 0), 1440);
	}

	private columnAt(clientX: number, fallback: number): number {
		const cols = this.options.columns();
		let nearest = fallback;
		let best = Infinity;
		for (let i = 0; i < cols.length; i++) {
			const rect = cols[i]?.getBoundingClientRect();
			if (!rect) continue;
			if (clientX >= rect.left && clientX < rect.right) return i;
			const distance = Math.min(Math.abs(clientX - rect.left), Math.abs(clientX - rect.right));
			if (distance < best) {
				best = distance;
				nearest = i;
			}
		}
		return nearest;
	}

	private onMove = (e: PointerEvent) => {
		const g = this.gesture;
		if (!g) return;
		if (!g.active) {
			if (Math.hypot(e.clientX - g.startX, e.clientY - g.startY) < DRAG_THRESHOLD_PX) return;
			g.active = true;
		}
		const deltaMinutes = ((e.clientY - g.startY) / this.options.slotHeight) * 60;
		if (g.mode === 'create') {
			this.preview = { column: g.column, range: drawRange(g.anchor, this.minuteAt(e.clientY, g.column)) };
		} else if (g.mode === 'move') {
			this.preview = {
				column: this.columnAt(e.clientX, g.column),
				range: moveRange(g.origin, deltaMinutes)
			};
		} else {
			const edge = g.mode === 'resize-top' ? 'top' : 'bottom';
			this.preview = { column: g.column, range: resizeRange(g.origin, edge, deltaMinutes) };
		}
	};

	private onUp = () => {
		const g = this.gesture;
		const preview = this.preview;
		this.finish();
		if (!g?.active) return;
		this.swallowClick();
		if (!preview) return;
		const day = this.options.days()[preview.column];
		if (!day) return;
		const start = atMinutes(day, preview.range.start);
		const end = atMinutes(day, preview.range.end);
		if (g.mode === 'create') {
			this.options.onCreate(start, end);
			return;
		}
		const unchanged =
			g.column === preview.column &&
			g.origin.start === preview.range.start &&
			g.origin.end === preview.range.end;
		if (g.event && !unchanged) this.options.onChange(g.event, start, end);
	};

	private cancel = () => {
		const wasActive = this.gesture?.active === true;
		this.finish();
		if (wasActive) this.swallowClick();
	};

	private onKey = (e: KeyboardEvent) => {
		if (e.key === 'Escape') this.cancel();
	};
}
