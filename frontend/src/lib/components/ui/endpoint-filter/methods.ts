export const METHOD_OPTIONS = [
	{ value: 'all', label: 'All methods' },
	{ value: 'get', label: 'GET' },
	{ value: 'post', label: 'POST' },
	{ value: 'put', label: 'PUT' },
	{ value: 'patch', label: 'PATCH' },
	{ value: 'delete', label: 'DELETE' },
	{ value: 'options', label: 'OPTIONS' },
	{ value: 'head', label: 'HEAD' }
] as const;

export type MethodFilterValue = (typeof METHOD_OPTIONS)[number]['value'];

export function isMethodFilterValue(value: string | null): value is MethodFilterValue {
	return METHOD_OPTIONS.some((option) => option.value === value);
}
