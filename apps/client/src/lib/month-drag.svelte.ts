import type { AgendaEvent } from '$lib/backend';
import { DRAG_THRESHOLD_PX } from '$lib/week-drag';

type Gesture = {
	event: AgendaEvent;
	grabDay: string;
	startX: number;
	startY: number;
	active: boolean;
};

type Options = {
	canEdit: (event: AgendaEvent) => boolean;
	onChange: (event: AgendaEvent, start: Date, end: Date) => void;
};

function dayOfPoint(x: number, y: number): string | null {
	for (const el of document.elementsFromPoint(x, y)) {
		const day = (el as HTMLElement).dataset?.day;
		if (day) return day;
	}
	return null;
}

function dayNumber(day: string): number {
	const [y, m, d] = day.split('-').map(Number);
	return Math.round(Date.UTC(y, m - 1, d) / 86400000);
}

function shiftDays(iso: string, days: number): Date {
	const d = new Date(iso);
	return new Date(d.getFullYear(), d.getMonth(), d.getDate() + days, d.getHours(), d.getMinutes(), 0, 0);
}

export class MonthDrag {
	gesture = $state<Gesture | null>(null);
	target = $state<string | null>(null);
	private options: Options;

	constructor(options: Options) {
		this.options = options;
	}

	get active(): boolean {
		return this.gesture?.active === true;
	}

	isDragging(event: AgendaEvent): boolean {
		return this.gesture?.active === true && this.gesture.event.id === event.id;
	}

	start = (e: PointerEvent, event: AgendaEvent) => {
		if (e.button !== 0 || e.pointerType === 'touch' || !this.options.canEdit(event)) return;
		const grabDay = dayOfPoint(e.clientX, e.clientY);
		if (!grabDay) return;
		e.stopPropagation();
		e.preventDefault();
		this.gesture = { event, grabDay, startX: e.clientX, startY: e.clientY, active: false };
		window.addEventListener('pointermove', this.onMove);
		window.addEventListener('pointerup', this.onUp);
		window.addEventListener('pointercancel', this.cancel);
		window.addEventListener('keydown', this.onKey);
	};

	private finish() {
		window.removeEventListener('pointermove', this.onMove);
		window.removeEventListener('pointerup', this.onUp);
		window.removeEventListener('pointercancel', this.cancel);
		window.removeEventListener('keydown', this.onKey);
		this.gesture = null;
		this.target = null;
	}

	private swallowClick() {
		const stop = (e: Event) => {
			e.stopPropagation();
			e.preventDefault();
		};
		window.addEventListener('click', stop, { capture: true, once: true });
		setTimeout(() => window.removeEventListener('click', stop, { capture: true }), 0);
	}

	private onMove = (e: PointerEvent) => {
		const g = this.gesture;
		if (!g) return;
		if (!g.active) {
			if (Math.hypot(e.clientX - g.startX, e.clientY - g.startY) < DRAG_THRESHOLD_PX) return;
			g.active = true;
		}
		this.target = dayOfPoint(e.clientX, e.clientY);
	};

	private onUp = () => {
		const g = this.gesture;
		const target = this.target;
		this.finish();
		if (!g?.active) return;
		this.swallowClick();
		if (!target || target === g.grabDay) return;
		const delta = dayNumber(target) - dayNumber(g.grabDay);
		this.options.onChange(g.event, shiftDays(g.event.start_at, delta), shiftDays(g.event.end_at, delta));
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
