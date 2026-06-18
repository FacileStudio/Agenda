const ADJECTIVES = [
	'bold', 'calm', 'cool', 'crisp', 'dark', 'deep', 'fair', 'fast',
	'fine', 'firm', 'free', 'glad', 'gold', 'good', 'gray', 'keen',
	'kind', 'late', 'lean', 'live', 'long', 'loud', 'mild', 'neat',
	'next', 'nice', 'open', 'pale', 'past', 'pure', 'rare', 'raw',
	'real', 'rich', 'ripe', 'safe', 'slim', 'slow', 'soft', 'sure',
	'tall', 'tidy', 'trim', 'true', 'vast', 'warm', 'wide', 'wild',
	'wise', 'aqua', 'blue', 'clay', 'dawn', 'dusk', 'east', 'edge',
	'glow', 'haze', 'jade', 'lush', 'mint', 'moon', 'navy', 'noon',
	'opal', 'peak', 'pine', 'plum', 'sand', 'silk', 'snow', 'star',
	'surf', 'teal', 'tide', 'vine', 'wave', 'west', 'zinc', 'ruby',
	'arch', 'airy', 'bone', 'core', 'dual', 'echo', 'even', 'flat',
	'full', 'half', 'high', 'iron', 'just', 'last', 'left', 'main',
	'new', 'odd', 'old', 'one', 'own', 'red', 'sea', 'shy',
	'sun', 'tan', 'top', 'two', 'dry', 'elm', 'fox', 'gem',
	'ice', 'ivy', 'jet', 'oak', 'orb', 'owl', 'pea', 'ram',
	'roe', 'rye', 'sky', 'sly', 'wax', 'yew', 'zen', 'ash',
];

const NOUNS = [
	'arch', 'barn', 'bell', 'bird', 'boat', 'bolt', 'cape', 'cave',
	'claw', 'cliff', 'cloud', 'coral', 'crane', 'creek', 'crow', 'dawn',
	'deer', 'dock', 'dove', 'drum', 'dune', 'eagle', 'elm', 'fawn',
	'fern', 'field', 'finch', 'flame', 'flint', 'ford', 'forge', 'frost',
	'gate', 'glen', 'grove', 'gust', 'hawk', 'haze', 'heath', 'heron',
	'hill', 'horn', 'isle', 'jade', 'lake', 'lark', 'leaf', 'ledge',
	'marsh', 'mesa', 'mill', 'mist', 'moss', 'nest', 'oak', 'opal',
	'orca', 'palm', 'path', 'peak', 'pine', 'plum', 'pond', 'quail',
	'rain', 'reef', 'ridge', 'ring', 'river', 'rock', 'rose', 'sage',
	'sail', 'sand', 'seal', 'shell', 'shore', 'slope', 'snow', 'spark',
	'stone', 'storm', 'swift', 'thorn', 'tide', 'tower', 'trail', 'vale',
	'vine', 'wave', 'wren', 'birch', 'bloom', 'bluff', 'brook', 'brush',
	'cedar', 'crest', 'delta', 'drift', 'falls', 'fjord', 'flume', 'glade',
	'haven', 'knoll', 'linen', 'lunar', 'maple', 'marsh', 'orbit', 'petal',
	'plume', 'prism', 'quartz', 'ridge', 'shade', 'slate', 'solar', 'spire',
	'steep', 'terra', 'torch', 'trove', 'vault', 'verge', 'wharf', 'woods',
];

export function roomName(uid: string): string {
	const hex = uid.split('@')[0].replace(/-/g, '');
	const a = parseInt(hex.slice(0, 4), 16) || 0;
	const b = parseInt(hex.slice(4, 8), 16) || 0;
	const c = parseInt(hex.slice(8, 10), 16) || 0;
	return `${ADJECTIVES[a % ADJECTIVES.length]}-${NOUNS[b % NOUNS.length]}-${c % 100}`;
}
