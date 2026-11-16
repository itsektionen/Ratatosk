<script lang="ts">
	import "#global.css";
	import favicon from "#lib/assets/favicon.svg";
	import { setAuth } from "#lib/auth.svelte.ts";
	import { onMount } from "svelte";
	import type { LayoutProps } from "./$types";

	let { children }: LayoutProps = $props();

	const auth = setAuth();
	onMount(() => auth.refresh());
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}
