<script>
	import { matrixService } from '$lib/stores/matrix.svelte';
	import { HouseIcon, MessageSquare, SettingsIcon } from '@lucide/svelte';
	import { Avatar, Navigation } from '@skeletonlabs/skeleton-svelte';
	const links = [
		{ label: 'Home', href: '/#', icon: HouseIcon },
		{ label: 'Rooms', href: '/#', icon: MessageSquare },
		{ label: 'Settings', href: '/#', icon: SettingsIcon }
	];

	let avatarSrc = matrixService.userProfile.avatarUrl;
	let avatarFallback = matrixService.userProfile.displayname.charAt(0).toUpperCase() || 'J';
</script>

<Navigation layout="bar">
	<Navigation.Menu class="grid grid-cols-4 gap-2">
		{#each links as link (link)}
			{@const Icon = link.icon}
			<Navigation.TriggerAnchor href={link.href}>
				<Icon class="size-5" />
				<Navigation.TriggerText>{link.label}</Navigation.TriggerText>
			</Navigation.TriggerAnchor>
		{/each}
		<Navigation.TriggerAnchor href="/" title="Your Profile" aria-label="Your Profile">
			<Avatar class="size-10">
				<Avatar.Image src={avatarSrc} alt="user's profile picture" />
				<Avatar.Fallback>{avatarFallback}</Avatar.Fallback>
			</Avatar>
			<Navigation.TriggerText>You</Navigation.TriggerText>
		</Navigation.TriggerAnchor>
	</Navigation.Menu>
</Navigation>
