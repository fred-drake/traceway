<script lang="ts">
	import * as Select from '$lib/components/ui/select';
	import { RootFilter } from '$lib/components/ui/root-filter';
	import { cn } from '$lib/utils.js';
	import { METHOD_OPTIONS } from './methods';

	type Props = {
		rootValue?: string;
		methodValue?: string;
		class?: string;
		onChange?: (value: string) => void;
	};

	let {
		rootValue = $bindable('all'),
		methodValue = $bindable('all'),
		class: className = 'sm:rounded-none sm:border-r-0',
		onChange
	}: Props = $props();

	const METHOD_COLORS: Record<string, string> = {
		GET: 'text-sky-600 dark:text-sky-400',
		POST: 'text-emerald-600 dark:text-emerald-400',
		PUT: 'text-amber-600 dark:text-amber-400',
		PATCH: 'text-orange-600 dark:text-orange-400',
		DELETE: 'text-rose-600 dark:text-rose-400',
		HEAD: 'text-violet-600 dark:text-violet-400',
		OPTIONS: 'text-violet-600 dark:text-violet-400'
	};

	const selectedMethod = $derived(
		METHOD_OPTIONS.find((option) => option.value === methodValue) ?? METHOD_OPTIONS[0]
	);
</script>

<Select.Root type="single" bind:value={methodValue} onValueChange={(v) => onChange?.(v ?? '')}>
	<Select.Trigger
		aria-label="Filter by HTTP method"
		class={cn('h-9 w-[130px] shrink-0 shadow-none', className)}
	>
		<span
			class={cn(
				selectedMethod.value !== 'all' && 'font-mono text-sm',
				METHOD_COLORS[selectedMethod.label]
			)}
		>
			{selectedMethod.label}
		</span>
	</Select.Trigger>
	<Select.Content>
		{#each METHOD_OPTIONS as option (option.value)}
			<Select.Item
				value={option.value}
				label={option.label}
				class={option.value === 'all' ? '' : 'font-mono text-sm'}
			>
				<span class={METHOD_COLORS[option.label] ?? ''}>{option.label}</span>
			</Select.Item>
		{/each}
	</Select.Content>
</Select.Root>

<RootFilter
	bind:value={rootValue}
	class={cn('h-9 w-[110px] shrink-0 shadow-none', className)}
	{onChange}
/>
