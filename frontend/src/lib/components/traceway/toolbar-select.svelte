<script lang="ts">
	import * as Select from '$lib/components/ui/select';
	import { cn } from '$lib/utils.js';

	type Option = { value: string; label: string };

	interface Props {
		value?: string;
		options: readonly Option[];
		class?: string;
		ariaLabel?: string;
		onChange?: (value: string) => void;
	}

	let { value = $bindable(''), options, class: className, ariaLabel, onChange }: Props = $props();

	const label = $derived(options.find((o) => o.value === value)?.label ?? options[0]?.label ?? '');
</script>

<Select.Root type="single" bind:value onValueChange={(v) => onChange?.(v ?? '')}>
	<Select.Trigger class={cn('h-9 w-[140px]', className)} aria-label={ariaLabel}>
		{label}
	</Select.Trigger>
	<Select.Content>
		{#each options as option (option.value)}
			<Select.Item value={option.value} label={option.label}>{option.label}</Select.Item>
		{/each}
	</Select.Content>
</Select.Root>
