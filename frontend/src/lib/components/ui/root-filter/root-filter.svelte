<script lang="ts">
	import ToolbarSelect from '$lib/components/traceway/toolbar-select.svelte';
	import { ROOT_FILTER_OPTIONS } from './options';

	type Props = {
		value?: string;
		class?: string;
		onChange?: (value: string) => void;
	};

	let { value = $bindable('all'), class: className = 'w-[120px]', onChange }: Props = $props();
</script>

<ToolbarSelect
	bind:value
	options={ROOT_FILTER_OPTIONS}
	class={className}
	ariaLabel="Filter by root or non-root"
	{onChange}
/>
