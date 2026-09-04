<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { Info } from '@lucide/svelte';
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
		children: Snippet;
	}

	let { class: className, children }: Props = $props();
</script>

<Alert.Root
	class={cn(
		'border-blue-200 bg-blue-50 text-blue-900 dark:border-blue-800 dark:bg-blue-950/50 dark:text-blue-200',
		className
	)}
>
	<Info class="text-blue-600 dark:text-blue-400" />
	<Alert.Description class="text-blue-800 dark:text-blue-300">
		{@render children()}
	</Alert.Description>
</Alert.Root>
