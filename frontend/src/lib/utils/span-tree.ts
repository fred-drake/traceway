import type { Span } from '$lib/types/spans';

export type SpanTreeRow = {
	span: Span;
	depth: number;
	hasChildren: boolean;
};

export function flattenSpanTree(
	spans: Span[],
	rootSpanId: string | undefined,
	collapsed: Set<string>
): SpanTreeRow[] {
	const byId = new Map(spans.map((s) => [s.id, s]));
	const childrenById = new Map<string, Span[]>();
	const roots: Span[] = [];

	for (const span of spans) {
		const parentId = span.parentSpanId;
		if (parentId && parentId !== rootSpanId && parentId !== span.id && byId.has(parentId)) {
			const siblings = childrenById.get(parentId);
			if (siblings) {
				siblings.push(span);
			} else {
				childrenById.set(parentId, [span]);
			}
		} else {
			roots.push(span);
		}
	}

	const byStartTime = (a: Span, b: Span) =>
		new Date(a.startTime).getTime() - new Date(b.startTime).getTime();

	const rows: SpanTreeRow[] = [];
	const visited = new Set<string>();

	function walk(span: Span, depth: number, hidden: boolean) {
		if (visited.has(span.id)) return;
		visited.add(span.id);
		const children = childrenById.get(span.id) ?? [];
		if (!hidden) {
			rows.push({ span, depth, hasChildren: children.length > 0 });
		}
		const hideChildren = hidden || collapsed.has(span.id);
		for (const child of [...children].sort(byStartTime)) {
			walk(child, depth + 1, hideChildren);
		}
	}

	for (const root of [...roots].sort(byStartTime)) {
		walk(root, 0, false);
	}

	for (const span of spans) {
		if (!visited.has(span.id)) {
			walk(span, 0, false);
		}
	}

	return rows;
}
