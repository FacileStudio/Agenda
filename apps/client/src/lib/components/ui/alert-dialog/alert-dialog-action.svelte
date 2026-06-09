<script lang="ts">
	import { AlertDialog as AlertDialogPrimitive } from "bits-ui";
	import { buttonVariants, type ButtonSize, type ButtonVariant } from "$lib/components/ui/button";
	import { cn } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		variant = "default",
		size = "default",
		...restProps
	}: AlertDialogPrimitive.ActionProps & {
		variant?: ButtonVariant;
		size?: ButtonSize;
	} = $props();
</script>

<AlertDialogPrimitive.Action
	bind:ref
	data-slot="alert-dialog-action"
	class={cn(
		buttonVariants({ variant, size }),
		variant === "destructive" && "border-destructive bg-destructive text-white hover:bg-destructive/90 hover:text-white",
		className
	)}
	{...restProps}
/>
