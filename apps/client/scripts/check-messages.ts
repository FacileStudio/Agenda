import { catalogs } from '../src/lib/i18n/messages';

const locales = ['fr', 'es', 'de'] as const;

function placeholders(message: unknown): string {
	const texts = typeof message === 'string' ? [message] : Object.values(message as Record<string, string>);
	const names = texts.flatMap((text) => [...text.matchAll(/\{(\w+)\}/g)].map((match) => match[1]));
	return [...new Set(names)].sort().join(',');
}

const problems: string[] = [];
for (const [key, english] of Object.entries(catalogs.en)) {
	for (const locale of locales) {
		const message = (catalogs[locale] as Record<string, unknown>)[key];
		if (!message) problems.push(`${locale} ${key}: empty`);
		else if (placeholders(message) !== placeholders(english)) problems.push(`${locale} ${key}: placeholders differ`);
	}
}

if (problems.length > 0) {
	console.error(problems.join('\n'));
	process.exit(1);
}
console.log(`${Object.keys(catalogs.en).length} messages, 4 locales, placeholders agree`);
