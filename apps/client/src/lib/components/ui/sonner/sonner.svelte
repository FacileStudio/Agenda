<script lang="ts">
	import { Toaster as SonnerToaster, type ToasterProps } from 'svelte-sonner';

	let { ...restProps }: ToasterProps = $props();
</script>

<SonnerToaster
	class="toaster group"
	position="bottom-right"
	toastOptions={{
		classes: {
			toast:
				'group toast group-[.toaster]:bg-popover group-[.toaster]:text-popover-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg group-[.toaster]:rounded-lg',
			description: 'group-[.toast]:text-muted-foreground',
			actionButton: 'group-[.toast]:bg-primary group-[.toast]:text-primary-foreground',
			cancelButton: 'group-[.toast]:bg-muted group-[.toast]:text-muted-foreground',
			error: 'group-[.toaster]:!bg-destructive group-[.toaster]:!text-white group-[.toaster]:!border-destructive',
			success: 'group-[.toaster]:!border-green-500/30'
		}
	}}
	{...restProps}
/>
