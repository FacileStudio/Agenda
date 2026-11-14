import { expect, mock, test } from 'bun:test';

mock.module('$app/environment', () => ({ browser: false }));
mock.module('$app/navigation', () => ({ goto: async () => {} }));

const { ApiError, notOwnSession } = await import('./backend');

const refusal = "only the person's own session can confirm actions";

test('only porte own-session refusal asks the person to sign in again', () => {
	expect(notOwnSession(new ApiError(403, 'permission_denied', refusal))).toBe(true);
	expect(notOwnSession(new ApiError(403, 'permission_denied', 'you cannot edit this calendar'))).toBe(false);
	expect(notOwnSession(new ApiError(403, 'csrf_failed', refusal))).toBe(false);
	expect(notOwnSession(new Error(refusal))).toBe(false);
});
