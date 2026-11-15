import { expect, test } from 'bun:test';
import { drawRange, moveRange, resizeRange } from './week-drag';

test('move snaps to 15 minutes and stays inside the day', () => {
	expect(moveRange({ start: 600, end: 660 }, 38)).toEqual({ start: 645, end: 705 });
	expect(moveRange({ start: 30, end: 90 }, -500)).toEqual({ start: 0, end: 60 });
	expect(moveRange({ start: 1380, end: 1440 }, 300)).toEqual({ start: 1380, end: 1440 });
});

test('resize keeps at least one slot and never crosses the other edge', () => {
	expect(resizeRange({ start: 600, end: 660 }, 'bottom', -200)).toEqual({ start: 600, end: 615 });
	expect(resizeRange({ start: 600, end: 660 }, 'top', 200)).toEqual({ start: 645, end: 660 });
	expect(resizeRange({ start: 600, end: 660 }, 'bottom', 29)).toEqual({ start: 600, end: 690 });
});

test('drawing works in both directions and fills the slots it touches', () => {
	expect(drawRange(605, 700)).toEqual({ start: 600, end: 705 });
	expect(drawRange(700, 605)).toEqual({ start: 600, end: 705 });
	expect(drawRange(600, 601)).toEqual({ start: 600, end: 615 });
	expect(drawRange(1439, 1439)).toEqual({ start: 1425, end: 1440 });
});
