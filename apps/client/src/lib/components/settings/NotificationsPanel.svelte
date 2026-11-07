<script lang="ts">
	/**
	 * Agenda has no mailer and no notification store — there is nothing here to switch on.
	 * Rather than render controls that persist nowhere, this tab states where reminders
	 * actually come from, which is the question people open it to answer.
	 */
	import { Alert, Button, NotificationsSection, SettingsRow, SettingsSection, StatusDot, icons } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
</script>

<div class="flex flex-col gap-10">
	<NotificationsSection title={t('notifications.reminders')} description={t('notifications.remindersText')}>
		<Alert tone="info">{t('notifications.alert')}</Alert>

		<SettingsRow label={t('notifications.client')} description={t('notifications.clientText')}>
			<StatusDot tone="success" label={t('notifications.supported')} />
		</SettingsRow>

		<SettingsRow label={t('notifications.emails')} description={t('notifications.emailsText')}>
			<StatusDot tone="neutral" label={t('notifications.none')} />
		</SettingsRow>

		<SettingsRow label={t('notifications.browser')} description={t('notifications.browserText')}>
			<StatusDot tone="neutral" label={t('notifications.notImplemented')} />
		</SettingsRow>
	</NotificationsSection>

	<SettingsSection title={t('notifications.where')} description={t('notifications.whereText')}>
		{#snippet actions()}
			<Button variant="outline" href="/settings/caldav" icon={icons.server}>
				{t('notifications.caldavButton')}
			</Button>
		{/snippet}

		<SettingsRow label={t('notifications.sharing')} description={t('notifications.sharingText')}>
			<Button variant="outline" href="/settings/members" icon={icons.usersGroup}>
				{t('notifications.sharingButton')}
			</Button>
		</SettingsRow>
	</SettingsSection>
</div>
