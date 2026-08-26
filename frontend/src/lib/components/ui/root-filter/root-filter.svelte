<script lang="ts">
	import ToolbarSelect from '$lib/components/traceway/toolbar-select.svelte';
	import { ROOT_FILTER_OPTIONS } from './options';

	type Props = {
		value?: string;
		class?: string;
	};

	let { value = $bindable('all'), class: className = 'w-[120px]' }: Props = $props();
</script>

<ToolbarSelect
	bind:value
	options={ROOT_FILTER_OPTIONS}
	class={className}
	ariaLabel="Filter by root or non-root"
/>
