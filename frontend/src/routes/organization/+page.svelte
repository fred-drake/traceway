<script lang="ts">
	import { page } from '$app/state';
	import { authState } from '$lib/state/auth.svelte';
	import { isBackendFramework, projectsState } from '$lib/state/projects.svelte';
	import { setTabParam } from '$lib/utils/url-params';
	import { gotoHref } from '$lib/utils/navigation';
	import { LoadingCircle } from '$lib/components/ui/loading-circle';
	import PageTabs from '$lib/components/traceway/page-tabs.svelte';
	import InfoCallout from '$lib/components/traceway/info-callout.svelte';
	import PageHeader from '$lib/components/traceway/page-header.svelte';
	import EmptyState from '$lib/components/traceway/empty-state.svelte';
	import OverviewTab from './overview-tab.svelte';
	import IssuesTab from './issues-tab.svelte';
	import MonitorsTab from './monitors-tab.svelte';
	import ProjectsTab from './projects-tab.svelte';

	const ALL_TABS = [
		{ value: 'overview', label: 'Overview' },
		{ value: 'issues', label: 'Issues' },
		{ value: 'monitors', label: 'Monitors' },
		{ value: 'projects', label: 'Projects' }
	];

	const TAB_DESCRIPTIONS: Record<string, string> = {
		issues: 'Recently active issues from every project, ordered by the last event received.',
		monitors:
			'All monitors from every project in the organization in one list, with their current status, uptime, and recent incidents.',
		projects:
			'Every project in the organization, with its framework and your effective access level.'
	};

	const orgs = $derived(authState.organizations);

	const currentOrganizationId = $derived.by(() => {
		const value = Number(page.url.searchParams.get('organizationId'));
		if (Number.isInteger(value) && orgs.some((organization) => organization.id === value)) {
			return value;
		}
		return orgs[0]?.id ?? null;
	});

	const currentOrganizationName = $derived(
		orgs.find((o) => o.id === currentOrganizationId)?.name ?? ''
	);

	const organizationProjects = $derived(
		currentOrganizationId === null
			? []
			: projectsState.projects.filter((project) => project.organizationId === currentOrganizationId)
	);

	const redirectHref = $derived.by(() => {
		if (currentOrganizationId === null) return null;
		if (organizationProjects.length === 1) return `/?projectId=${organizationProjects[0].id}`;
		return null;
	});

	const hasBackendProjects = $derived(
		organizationProjects.some((project) => isBackendFramework(project.framework))
	);

	const tabs = $derived(
		hasBackendProjects ? ALL_TABS : ALL_TABS.filter((tab) => tab.value !== 'monitors')
	);

	const activeTab = $derived.by(() => {
		const tab = page.url.searchParams.get('tab') || 'overview';
		return tabs.some((t) => t.value === tab) ? tab : 'overview';
	});

	function setTab(tab: string) {
		setTabParam(tab);
	}

	$effect(() => {
		if (!projectsState.loaded) return;
		const tab = page.url.searchParams.get('tab');
		if (tab && !tabs.some((t) => t.value === tab)) {
			setTabParam('overview');
		}
	});

	$effect(() => {
		const organizationId = currentOrganizationId;
		if (organizationId === null) return;
		if (projectsState.loaded && redirectHref) {
			gotoHref(redirectHref, { replaceState: true });
			return;
		}
		if (
			page.url.searchParams.get('organizationId') === String(organizationId) &&
			!page.url.searchParams.has('projectId')
		) {
			return;
		}
		const url = new URL(page.url);
		url.searchParams.set('organizationId', String(organizationId));
		url.searchParams.delete('projectId');
		gotoHref(url.pathname + url.search, {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
	});
</script>

{#if !projectsState.loaded || redirectHref}
	<div class="flex h-48 items-center justify-center">
		<LoadingCircle size="xlg" />
	</div>
{:else}
	<div class="space-y-4">
		<PageHeader
			title={currentOrganizationName || 'Organization'}
			description={hasBackendProjects
				? 'Live instance health, active response, and telemetry across every project.'
				: 'Issues and active response across every project.'}
		/>

		<PageTabs {tabs} {activeTab} onTabChange={setTab} />

		{#if TAB_DESCRIPTIONS[activeTab]}
			<InfoCallout>{TAB_DESCRIPTIONS[activeTab]}</InfoCallout>
		{/if}

		{#if currentOrganizationId === null}
			<EmptyState message="You are not a member of any organization yet." />
		{:else if organizationProjects.length === 0}
			<EmptyState message="This organization has no projects yet." />
		{:else if activeTab === 'overview'}
			<OverviewTab organizationId={currentOrganizationId} {hasBackendProjects} />
		{:else if activeTab === 'issues'}
			<IssuesTab organizationId={currentOrganizationId} />
		{:else if activeTab === 'monitors'}
			<MonitorsTab organizationId={currentOrganizationId} />
		{:else if activeTab === 'projects'}
			<ProjectsTab organizationId={currentOrganizationId} />
		{/if}
	</div>
{/if}
