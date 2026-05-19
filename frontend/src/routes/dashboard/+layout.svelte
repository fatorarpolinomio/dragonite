<script lang="ts">
    import { page } from '$app/state';
	import NavigationBarView from '$lib/components/NavigationBarView.svelte';
	import NavigationSideBarView from '$lib/components/NavigationSideBarView.svelte';

	let { children } = $props();
    let isInRoom = $derived(page.url.pathname.startsWith('/dashboard/rooms/'));
</script>

<div class="grid h-screen w-full grid-cols-[auto_1fr] overflow-hidden border border-surface-200-800 max-lg:grid-cols-1 max-lg:grid-rows-[1fr_auto]">
    <header class="max-lg:fixed max-lg:right-0 max-lg:bottom-0 max-lg:left-0 max-lg:order-2 max-lg:col-span-full">
        <nav class="hidden h-full lg:block">
            <NavigationSideBarView />
        </nav>
        {#if !isInRoom}
            <nav class="h-full lg:hidden">
                <NavigationBarView />
            </nav>
        {/if}
    </header>

    <div class="h-full overflow-hidden max-lg:order-1 max-lg:col-span-full {isInRoom ? '' : 'max-lg:pb-16'}">
        {@render children()}
    </div>
</div>
