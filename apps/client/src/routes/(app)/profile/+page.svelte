<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { backend, type UserProfile, type MailAccount } from '$lib/backend';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { toast } from 'svelte-sonner';
	import { User, Mail, PenLine, Plus, Trash2, Plug, X, Save, FileText, Pencil } from 'lucide-svelte';
	import type { EmailTemplate } from '$lib/backend';

	const app = getContext<{
		user: UserProfile | null;
		setUser: (user: UserProfile) => void;
		refreshAccounts: () => Promise<void>;
	}>('app');

	let activeTab = $state<'profile' | 'accounts' | 'signatures' | 'templates'>('profile');

	const tabs = [
		{ id: 'profile' as const, label: 'Profile', icon: User },
		{ id: 'accounts' as const, label: 'Accounts', icon: Mail },
		{ id: 'signatures' as const, label: 'Signatures', icon: PenLine },
		{ id: 'templates' as const, label: 'Templates', icon: FileText }
	];

	let accounts = $state<MailAccount[]>([]);
	let showForm = $state(false);
	let saving = $state(false);
	let testing = $state(false);
	let deleting = $state<number | null>(null);
	let signatures = $state<Record<number, string>>({});
	let savingSignature = $state<number | null>(null);

	let templates = $state<EmailTemplate[]>([]);
	let showTemplateForm = $state(false);
	let editingTemplate = $state<EmailTemplate | null>(null);
	let templateName = $state('');
	let templateSubject = $state('');
	let templateBody = $state('');
	let savingTemplate = $state(false);
	let deletingTemplate = $state<number | null>(null);

	let name = $state('');
	let email = $state('');
	let imapHost = $state('');
	let imapPort = $state(993);
	let imapUser = $state('');
	let imapPassword = $state('');
	let smtpHost = $state('');
	let smtpPort = $state(587);
	let smtpUser = $state('');
	let smtpPassword = $state('');

	function resetForm() {
		name = '';
		email = '';
		imapHost = '';
		imapPort = 993;
		imapUser = '';
		imapPassword = '';
		smtpHost = '';
		smtpPort = 587;
		smtpUser = '';
		smtpPassword = '';
		showForm = false;
	}

	async function loadAccounts() {
		const result = await backend.listAccounts();
		accounts = result.accounts;
		for (const account of accounts) {
			if (!(account.id in signatures)) {
				signatures[account.id] = account.signature || '';
			}
		}
	}

	async function saveSignature(accountId: number) {
		savingSignature = accountId;
		try {
			await backend.updateAccount(accountId, { signature: signatures[accountId] });
			toast.success('Signature saved');
			await app.refreshAccounts();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save signature');
		}
		savingSignature = null;
	}

	async function testConnection() {
		testing = true;
		try {
			await backend.testConnection({
				imap_host: imapHost,
				imap_port: imapPort,
				imap_user: imapUser,
				imap_password: imapPassword,
				smtp_host: smtpHost,
				smtp_port: smtpPort,
				smtp_user: smtpUser,
				smtp_password: smtpPassword
			});
			toast.success('Connection test passed');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Connection test failed');
		}
		testing = false;
	}

	async function addAccount() {
		if (!name || !email || !imapHost || !smtpHost) return;
		saving = true;
		try {
			await backend.createAccount({
				name,
				email,
				imap_host: imapHost,
				imap_port: imapPort,
				imap_user: imapUser,
				imap_password: imapPassword,
				smtp_host: smtpHost,
				smtp_port: smtpPort,
				smtp_user: smtpUser,
				smtp_password: smtpPassword,
				signature: '',
				is_default: accounts.length === 0
			});
			toast.success('Account added');
			resetForm();
			await loadAccounts();
			await app.refreshAccounts();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to add account');
		}
		saving = false;
	}

	async function deleteAccount(id: number) {
		deleting = id;
		try {
			await backend.deleteAccount(id);
			toast.success('Account deleted');
			await loadAccounts();
			await app.refreshAccounts();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete account');
		}
		deleting = null;
	}

	function getInitials(value: string) {
		const parts = value.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase();
	}

	function displayName(user: UserProfile | null) {
		return user?.name?.trim() || user?.email || '';
	}

	async function loadTemplates() {
		try {
			const result = await backend.listTemplates();
			templates = result.templates;
		} catch {
			templates = [];
		}
	}

	function resetTemplateForm() {
		templateName = '';
		templateSubject = '';
		templateBody = '';
		showTemplateForm = false;
		editingTemplate = null;
	}

	function startEditTemplate(tmpl: EmailTemplate) {
		editingTemplate = tmpl;
		templateName = tmpl.name;
		templateSubject = tmpl.subject;
		templateBody = tmpl.body_text || tmpl.body_html;
		showTemplateForm = true;
	}

	async function saveTemplate() {
		if (!templateName.trim()) return;
		savingTemplate = true;
		try {
			const data = {
				name: templateName,
				subject: templateSubject,
				body_html: templateBody,
				body_text: templateBody.replace(/<[^>]*>/g, '')
			};
			if (editingTemplate) {
				await backend.updateTemplate(editingTemplate.id, data);
				toast.success('Template updated');
			} else {
				await backend.createTemplate(data);
				toast.success('Template created');
			}
			resetTemplateForm();
			await loadTemplates();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save template');
		}
		savingTemplate = false;
	}

	async function deleteTemplate(id: number) {
		deletingTemplate = id;
		try {
			await backend.deleteTemplate(id);
			toast.success('Template deleted');
			await loadTemplates();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete template');
		}
		deletingTemplate = null;
	}

	onMount(() => {
		loadAccounts();
		loadTemplates();
	});
</script>

<svelte:head>
	<title>Settings — Courrier</title>
</svelte:head>

<div class="flex flex-col gap-0 h-full">
	<div>
		<div class="px-6 pt-6 pb-0">
			<h1 class="text-2xl font-semibold">Settings</h1>
			<p class="text-sm text-muted-foreground mt-1">Manage your account and mail settings.</p>
		</div>
		<div class="flex items-center gap-1 px-6 pt-4 pb-0">
			{#each tabs as tab}
				<button
					class="inline-flex items-center gap-2 rounded-md px-4 h-9 text-sm transition-colors {activeTab === tab.id
						? 'bg-foreground text-background font-medium'
						: 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
					onclick={() => (activeTab = tab.id)}
				>
					<tab.icon class="h-4 w-4" />
					{tab.label}
				</button>
			{/each}
		</div>
	</div>

	<div class="flex-1 overflow-auto p-6">
		<div class="max-w-2xl">
			{#if activeTab === 'profile'}
				<div class="space-y-6">
					<div class="flex items-center gap-4">
						{#if app.user?.avatar_url}
							<img
								src={app.user.avatar_url}
								alt={displayName(app.user)}
								class="h-24 w-24 rounded-full border border-border object-cover"
							/>
						{:else}
							<div
								class="flex h-24 w-24 items-center justify-center rounded-full border border-border bg-foreground text-2xl font-semibold text-background"
							>
								{getInitials(displayName(app.user))}
							</div>
						{/if}
						{#if app.user?.avatar_source === 'oidc'}
							<p class="text-xs text-muted-foreground">Avatar synced from SSO</p>
						{/if}
					</div>

					<div class="space-y-4">
						<div class="space-y-2">
							<Label for="profile-name">Name</Label>
							<Input id="profile-name" value={app.user?.name ?? ''} disabled />
						</div>

						<div class="space-y-2">
							<Label for="profile-email">Email</Label>
							<Input id="profile-email" value={app.user?.email ?? ''} disabled />
						</div>
					</div>
				</div>
			{:else if activeTab === 'accounts'}
				<div class="space-y-4">
					{#if accounts.length === 0 && !showForm}
						<p class="text-sm text-muted-foreground">No mail accounts configured yet.</p>
					{:else}
						<div class="space-y-0">
							{#each accounts as account, i}
								<div
									class="flex items-center justify-between py-3 {i < accounts.length - 1
										? 'border-b'
										: ''}"
								>
									<div class="min-w-0">
										<p class="font-medium text-sm">{account.name}</p>
										<p class="text-xs text-muted-foreground">{account.email}</p>
										<p class="text-xs text-muted-foreground/60 mt-0.5">{account.imap_host}</p>
									</div>
									<div class="flex items-center gap-2 shrink-0">
										{#if account.is_default}
											<span
												class="text-xs bg-foreground text-background px-2 py-0.5 rounded-full"
												>Default</span
											>
										{/if}
										<Button
											variant="ghost"
											size="icon"
											class="h-8 w-8 text-muted-foreground hover:text-destructive"
											onclick={() => deleteAccount(account.id)}
											disabled={deleting === account.id}
										>
											<Trash2 class="h-4 w-4" />
										</Button>
									</div>
								</div>
							{/each}
						</div>
					{/if}

					{#if showForm}
						<div class="space-y-4 pt-2">
							<div class="grid grid-cols-2 gap-4">
								<div class="space-y-2">
									<Label for="acc-name">Display name</Label>
									<Input id="acc-name" bind:value={name} placeholder="Work Email" />
								</div>
								<div class="space-y-2">
									<Label for="acc-email">Email address</Label>
									<Input
										id="acc-email"
										type="email"
										bind:value={email}
										placeholder="you@example.com"
									/>
								</div>
							</div>

							<div class="pt-2">
								<p class="text-sm font-medium mb-3">IMAP (Incoming)</p>
								<div class="grid grid-cols-2 gap-4">
									<div class="space-y-2">
										<Label for="imap-host">Host</Label>
										<Input
											id="imap-host"
											bind:value={imapHost}
											placeholder="imap.example.com"
										/>
									</div>
									<div class="space-y-2">
										<Label for="imap-port">Port</Label>
										<Input id="imap-port" type="number" bind:value={imapPort} />
									</div>
									<div class="space-y-2">
										<Label for="imap-user">Username</Label>
										<Input
											id="imap-user"
											bind:value={imapUser}
											placeholder="you@example.com"
										/>
									</div>
									<div class="space-y-2">
										<Label for="imap-password">Password</Label>
										<Input
											id="imap-password"
											type="password"
											bind:value={imapPassword}
										/>
									</div>
								</div>
							</div>

							<div class="pt-2">
								<p class="text-sm font-medium mb-3">SMTP (Outgoing)</p>
								<div class="grid grid-cols-2 gap-4">
									<div class="space-y-2">
										<Label for="smtp-host">Host</Label>
										<Input
											id="smtp-host"
											bind:value={smtpHost}
											placeholder="smtp.example.com"
										/>
									</div>
									<div class="space-y-2">
										<Label for="smtp-port">Port</Label>
										<Input id="smtp-port" type="number" bind:value={smtpPort} />
									</div>
									<div class="space-y-2">
										<Label for="smtp-user">Username</Label>
										<Input
											id="smtp-user"
											bind:value={smtpUser}
											placeholder="you@example.com"
										/>
									</div>
									<div class="space-y-2">
										<Label for="smtp-password">Password</Label>
										<Input
											id="smtp-password"
											type="password"
											bind:value={smtpPassword}
										/>
									</div>
								</div>
							</div>

							<div class="flex items-center gap-2 pt-2">
								<Button
									variant="outline"
									size="sm"
									class="gap-1.5"
									onclick={testConnection}
									disabled={testing || !imapHost}
								>
									<Plug class="h-4 w-4" />
									{testing ? 'Testing...' : 'Test Connection'}
								</Button>
								<div class="flex-1"></div>
								<Button variant="ghost" size="sm" class="gap-1.5" onclick={resetForm}>
									<X class="h-4 w-4" />
									Cancel
								</Button>
								<Button
									size="sm"
									class="gap-1.5"
									onclick={addAccount}
									disabled={saving || !name || !email || !imapHost || !smtpHost}
								>
									<Plus class="h-4 w-4" />
									{saving ? 'Saving...' : 'Add Account'}
								</Button>
							</div>
						</div>
					{:else}
						<Button
							variant="outline"
							size="sm"
							class="gap-1.5"
							onclick={() => (showForm = true)}
						>
							<Plus class="h-4 w-4" />
							Add Account
						</Button>
					{/if}
				</div>
			{:else if activeTab === 'signatures'}
				<div class="space-y-6">
					{#if accounts.length === 0}
						<p class="text-sm text-muted-foreground">
							Add a mail account first to configure signatures.
						</p>
					{:else}
						{#each accounts as account}
							<div class="space-y-2">
								<div class="flex items-center gap-2">
									<p class="text-sm font-medium">{account.name}</p>
									<span class="text-xs text-muted-foreground">{account.email}</span>
								</div>
								<textarea
									bind:value={signatures[account.id]}
									placeholder="Write your email signature..."
									class="h-24 w-full resize-none rounded-md border border-input bg-transparent px-3 py-2 text-sm leading-relaxed outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 placeholder:text-muted-foreground"
								></textarea>
								<Button
									variant="outline"
									size="sm"
									class="gap-1.5"
									onclick={() => saveSignature(account.id)}
									disabled={savingSignature === account.id}
								>
									<Save class="h-4 w-4" />
									{savingSignature === account.id ? 'Saving...' : 'Save'}
								</Button>
							</div>
						{/each}
					{/if}
				</div>
			{:else if activeTab === 'templates'}
				<div class="space-y-4">
					{#if templates.length === 0 && !showTemplateForm}
						<p class="text-sm text-muted-foreground">No templates yet. Create one to speed up composing.</p>
					{:else if !showTemplateForm}
						<div class="space-y-0">
							{#each templates as tmpl, i}
								<div class="flex items-center justify-between py-3 {i < templates.length - 1 ? 'border-b' : ''}">
									<div class="min-w-0">
										<p class="font-medium text-sm">{tmpl.name}</p>
										{#if tmpl.subject}
											<p class="text-xs text-muted-foreground">{tmpl.subject}</p>
										{/if}
									</div>
									<div class="flex items-center gap-1 shrink-0">
										<Button
											variant="ghost"
											size="icon"
											class="h-8 w-8"
											onclick={() => startEditTemplate(tmpl)}
										>
											<Pencil class="h-4 w-4" />
										</Button>
										<Button
											variant="ghost"
											size="icon"
											class="h-8 w-8 text-muted-foreground hover:text-destructive"
											onclick={() => deleteTemplate(tmpl.id)}
											disabled={deletingTemplate === tmpl.id}
										>
											<Trash2 class="h-4 w-4" />
										</Button>
									</div>
								</div>
							{/each}
						</div>
					{/if}

					{#if showTemplateForm}
						<div class="space-y-4 pt-2">
							<div class="space-y-2">
								<Label for="tmpl-name">Template name</Label>
								<Input id="tmpl-name" bind:value={templateName} placeholder="e.g. Welcome email" />
							</div>
							<div class="space-y-2">
								<Label for="tmpl-subject">Subject line</Label>
								<Input id="tmpl-subject" bind:value={templateSubject} placeholder="Subject" />
							</div>
							<div class="space-y-2">
								<Label for="tmpl-body">Body</Label>
								<textarea
									id="tmpl-body"
									bind:value={templateBody}
									placeholder="Write your template content..."
									class="h-40 w-full resize-none rounded-md border border-input bg-transparent px-3 py-2 text-sm leading-relaxed outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 placeholder:text-muted-foreground"
								></textarea>
							</div>
							<div class="flex items-center gap-2">
								<div class="flex-1"></div>
								<Button variant="ghost" size="sm" class="gap-1.5" onclick={resetTemplateForm}>
									<X class="h-4 w-4" />
									Cancel
								</Button>
								<Button
									size="sm"
									class="gap-1.5"
									onclick={saveTemplate}
									disabled={savingTemplate || !templateName.trim()}
								>
									<Save class="h-4 w-4" />
									{savingTemplate ? 'Saving...' : editingTemplate ? 'Update' : 'Create'}
								</Button>
							</div>
						</div>
					{:else}
						<Button
							variant="outline"
							size="sm"
							class="gap-1.5"
							onclick={() => (showTemplateForm = true)}
						>
							<Plus class="h-4 w-4" />
							New Template
						</Button>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</div>
