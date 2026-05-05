<script>
	import {
		ArrowLeftRightIcon,
		HouseIcon,
		LogOut,
		MessageSquare,
		SettingsIcon
	} from '@lucide/svelte';
	import { Navigation } from '@skeletonlabs/skeleton-svelte';
	import UserSearchBar from '$lib/components/UserSearchBar.svelte';

	let isLayoutRail = $state(true);

	function toggleLayout() {
		isLayoutRail = !isLayoutRail;
	}

	const links = [
		{ label: 'Home', href: '/#', icon: HouseIcon },
		{ label: 'Rooms', href: '/#', icon: MessageSquare },
		{ label: 'Settings', href: '/#', icon: SettingsIcon }
	];
</script>

<Navigation layout={isLayoutRail ? 'rail' : 'sidebar'} class="grid grid-rows-[auto_1fr_auto] gap-4">
	<Navigation.Header>
		<Navigation.Trigger onclick={toggleLayout}>
			<ArrowLeftRightIcon class={isLayoutRail ? 'size-5' : 'size-4'} />
			{#if !isLayoutRail}<span>Resize</span>{/if}
		</Navigation.Trigger>
	</Navigation.Header>
	<Navigation.Content class="overflow-visible">
		<Navigation.Menu>
			<UserSearchBar isRail={isLayoutRail} onExpand={() => (isLayoutRail = false)} />
			{#each links as link (link)}
				{@const Icon = link.icon}
				<Navigation.TriggerAnchor>
					<Icon class={isLayoutRail ? 'size-5' : 'size-4'} />
					<Navigation.TriggerText>{link.label}</Navigation.TriggerText>
				</Navigation.TriggerAnchor>
			{/each}
		</Navigation.Menu>
	</Navigation.Content>
	<Navigation.Footer>
		<Navigation.TriggerAnchor href="/" title="Settings" aria-label="Settings">
			<LogOut class="size-4" />
			<Navigation.TriggerText>Leave</Navigation.TriggerText>
		</Navigation.TriggerAnchor>
	</Navigation.Footer>
</Navigation>
