export const ROOT_FILTER_OPTIONS = [
	{ value: 'all', label: 'All' },
	{ value: 'root', label: 'Root' },
	{ value: 'non_root', label: 'Non-root' }
] as const;

export type RootFilterValue = (typeof ROOT_FILTER_OPTIONS)[number]['value'];

export function isRootFilterValue(value: string | null): value is RootFilterValue {
	return ROOT_FILTER_OPTIONS.some((option) => option.value === value);
}
