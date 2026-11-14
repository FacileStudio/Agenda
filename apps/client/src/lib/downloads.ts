import type { CliCommand } from '@facile/muse';
import { t } from '$lib/i18n/index.svelte';

const CLI_REPO = 'https://github.com/FacileStudio/agenda-cli';

/** The agenda CLI's releases page. */
export const RELEASES_HREF = `${CLI_REPO}/releases`;

/** The checksum file of the latest agenda CLI release. */
export const CHECKSUMS_HREF = `${CLI_REPO}/releases/latest/download/checksums.txt`;

/** The install commands of the agenda CLI, labelled in the current locale. */
export function cliCommands(): CliCommand[] {
	return [
		{ label: t('landing.download.facile'), command: 'facile install agenda' },
		{ label: t('landing.download.script'), command: `curl -fsSL https://raw.githubusercontent.com/FacileStudio/agenda-cli/main/install.sh | bash` }
	];
}
