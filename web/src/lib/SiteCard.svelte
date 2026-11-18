<script lang="ts">
	import Button from "#lib/Button.svelte";
	import { getEditor } from "#lib/editor.svelte.ts";
	import UpdateIcon from "~icons/ph/pencil-simple-fill";
	import DeleteIcon from "~icons/ph/trash-fill";

	let {
		subdomain,
		url,
		author,
		onupdate,
		ondelete,
	}: {
		subdomain: string;
		url: string;
		author?: string;
		onupdate?: () => void;
		ondelete?: () => void;
	} = $props();

	const editor = getEditor();

	// Pages render at this size and get scaled down to fit
	const PREVIEW_W = 512;
	const PREVIEW_H = (PREVIEW_W * 9) / 16;

	let previewWidth = $state(PREVIEW_W);
	let loaded = $state(false);
	let active = $derived(editor.isActive(subdomain));
</script>

<div
	class={[
		"group relative flex flex-col border hover:border-primary",
		active ? "border-primary" : "border-border",
	]}
>
	<div
		class={[
			"flex items-center gap-2 border-b p-1.5 group-hover:border-primary",
			active ? "border-primary" : "border-border",
		]}
	>
		<a
			href={url}
			target="_blank"
			rel="noreferrer"
			class="flex min-w-0 flex-1 items-baseline gap-2 px-1 text-foreground no-underline after:absolute after:inset-0"
		>
			<span class="truncate text-sm leading-6 font-medium">{subdomain}</span>
			{#if author}
				<span class="truncate text-xs text-muted-foreground">by {author}</span>
			{/if}
		</a>
		{#if onupdate}
			<Button
				onclick={onupdate}
				title="Update"
				aria-label="Update"
				class="relative z-10 size-6 text-sm"
			>
				<UpdateIcon />
			</Button>
		{/if}
		{#if ondelete}
			<Button
				onclick={ondelete}
				title="Delete"
				aria-label="Delete"
				variant="danger"
				class="relative z-10 size-6 text-sm"
			>
				<DeleteIcon />
			</Button>
		{/if}
	</div>
	<div class="aspect-video min-w-0 overflow-hidden dash-shade-2" bind:clientWidth={previewWidth}>
		<iframe
			src={url}
			title="Preview of {subdomain}"
			loading="lazy"
			// No allow-same-origin, so a site can't touch our cookies. The server allows it to fetch its own files
			sandbox="allow-scripts"
			width={PREVIEW_W}
			height={PREVIEW_H}
			style:transform="scale({previewWidth / PREVIEW_W})"
			onload={() => (loaded = true)}
			class={[
				"pointer-events-none origin-top-left border-0 bg-white transition-opacity duration-300",
				loaded ? "opacity-100" : "opacity-0",
			]}
		></iframe>
	</div>
</div>
