<script lang="ts">
	import "#global.css";
	import favicon from "#lib/assets/favicon.svg";
	import { setAuth } from "#lib/auth.svelte.ts";
	import { loadBoot } from "#lib/boot.ts";
	import { setSites } from "#lib/sites.svelte.ts";
	import { onMount } from "svelte";
	import type { LayoutProps } from "./$types";

	let { children }: LayoutProps = $props();

	const auth = setAuth();
	const sites = setSites();

	onMount(async () => {
		try {
			const boot = await loadBoot();
			auth.apply(boot);
			sites.list = boot.sites;
			sites.baseDomain = boot.baseDomain;
		} catch {
			auth.user = null;
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}
