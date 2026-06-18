import { browser } from '$app/environment';

const STORAGE_KEY = 'agenda:space-context';

export type SpaceInfo = {
	spaceId: number;
	name: string;
	role: string;
};

export type SpaceContext = 'personal' | SpaceInfo;

function load(): SpaceContext {
	if (!browser) return 'personal';
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return 'personal';
		const parsed = JSON.parse(raw);
		if (parsed === 'personal') return 'personal';
		if (parsed && typeof parsed.spaceId === 'number') return parsed as SpaceInfo;
	} catch {}
	return 'personal';
}

function save(ctx: SpaceContext) {
	if (!browser) return;
	localStorage.setItem(STORAGE_KEY, JSON.stringify(ctx));
}

let current = $state<SpaceContext>(load());

export function getSpaceContext(): SpaceContext {
	return current;
}

export function setSpaceContext(ctx: SpaceContext) {
	current = ctx;
	save(ctx);
}

export function isPersonal(): boolean {
	return current === 'personal';
}

export function canManage(): boolean {
	if (current === 'personal') return false;
	return current.role === 'owner' || current.role === 'admin';
}

export function spaceId(): number | null {
	if (current === 'personal') return null;
	return current.spaceId;
}

export function spaceRole(): string | null {
	if (current === 'personal') return null;
	return current.role;
}
