<script lang="ts">
	import { TriangleAlert } from '@lucide/svelte';

	let {
		message = 'Some projects could not be read. The data shown is still current.'
	}: { message?: string } = $props();
</script>

<div
	class="flex items-center gap-2 rounded-lg border border-amber-500/25 bg-amber-500/5 px-4 py-2.5 text-sm text-amber-700 dark:text-amber-300"
>
	<TriangleAlert class="size-4 shrink-0" />
	{message}
</div>
