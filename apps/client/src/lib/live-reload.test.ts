import { expect, test } from 'bun:test';
import { requestReload, setViewBusy, settleReloads } from './live-reload';

test('a reload waits while the view is busy and runs once when it settles', () => {
	let busy = true;
	let runs = 0;
	const reload = () => runs++;
	const undo = setViewBusy(() => busy);

	requestReload(reload);
	requestReload(reload);
	settleReloads();
	expect(runs).toBe(0);

	busy = false;
	settleReloads();
	expect(runs).toBe(1);

	requestReload(reload);
	expect(runs).toBe(2);
	undo();
});

test('undoing the busy check flushes what waited', () => {
	let runs = 0;
	const undo = setViewBusy(() => true);
	requestReload(() => runs++);
	undo();
	expect(runs).toBe(1);
});
