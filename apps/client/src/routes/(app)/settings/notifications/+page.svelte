<script lang="ts">
	/**
	 * Agenda has no mailer and no notification store — there is nothing here to switch on.
	 * Rather than render controls that persist nowhere, this tab states where reminders
	 * actually come from, which is the question people open it to answer.
	 */
	import { Alert, Button, SettingsRow, SettingsSection, StatusDot, icons } from '@facile/muse';
</script>

<svelte:head>
	<title>Notifications — Agenda</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<SettingsSection
		title="Rappels"
		description="Agenda ne notifie pas lui-même. Il publie vos événements, et c’est votre client qui sonne."
	>
		<Alert tone="info">
			Un rappel voyage dans l’événement, pas à côté : Agenda conserve l’ICS d’origine
			(<span class="font-fc-mono">events.raw_ics</span>) et le rend tel quel en CalDAV, alarmes
			comprises. Réglez donc vos rappels dans Calendrier iOS, Calendrier macOS, Thunderbird ou
			DAVx⁵ — ils remontent ici et repartent vers vos autres appareils.
		</Alert>

		<SettingsRow
			label="Rappels dans le client"
			description="Les VALARM de vos événements sont conservés à l’octet près lors des allers-retours CalDAV."
		>
			<StatusDot tone="success" label="Pris en charge" />
		</SettingsRow>

		<SettingsRow
			label="E-mails"
			description="Cette instance n’a pas de relais SMTP configuré : aucune invitation ni aucun rappel n’est envoyé par mail."
		>
			<StatusDot tone="neutral" label="Aucun envoi" />
		</SettingsRow>

		<SettingsRow
			label="Notifications navigateur"
			description="Agenda ne demande pas la permission Notifications et n’enregistre pas de service worker push."
		>
			<StatusDot tone="neutral" label="Non implémenté" />
		</SettingsRow>
	</SettingsSection>

	<SettingsSection
		title="Où régler ses rappels"
		description="Connectez d’abord un client CalDAV, puis réglez les alarmes dedans."
	>
		{#snippet actions()}
			<Button variant="outline" href="/settings/api" icon={icons.server}>Connexion CalDAV</Button>
		{/snippet}

		<SettingsRow
			label="Partage d’événement"
			description="Les participants d’un événement sont enregistrés, mais ils ne reçoivent pas d’invitation : partagez le calendrier plutôt que l’événement."
		>
			<Button variant="outline" href="/settings/members" icon={icons.usersGroup}>
				Partage des calendriers
			</Button>
		</SettingsRow>
	</SettingsSection>
</div>
