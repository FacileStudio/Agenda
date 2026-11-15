import type { MenuEntry } from '@facile/muse';

class ContextMenuState {
	open = $state(false);
	x = $state(0);
	y = $state(0);
	items = $state<MenuEntry[]>([]);

	show(e: MouseEvent, items: MenuEntry[]) {
		e.preventDefault();
		e.stopPropagation();
		this.x = e.clientX;
		this.y = e.clientY;
		this.items = items;
		this.open = true;
	}

	hide = () => {
		this.open = false;
	};
}

export const contextMenu = new ContextMenuState();
