<script lang="ts">
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
		author: string;
		onupdate?: () => void;
		ondelete?: () => void;
	} = $props();

	const editor = getEditor();

	let loaded = $state(false);
	let active = $derived(editor.isActive(subdomain));
</script>

<div class={["flex flex-col border", active ? "border-primary" : "border-border"]}>
	<div class="aspect-video min-w-0 dash-shade-2">
		<iframe
			src={url}
			title="Preview of {subdomain}"
			loading="lazy"
			sandbox=""
			onload={() => (loaded = true)}
			class={[
				"size-full border-0 bg-white transition-opacity duration-300",
				loaded ? "opacity-100" : "opacity-0",
			]}
		></iframe>
	</div>
	<hr class={active ? "border-primary" : "border-border"} />
	<div class="flex items-center justify-between gap-2 p-2">
		<div class="min-w-0">
			<a href={url} target="_blank" rel="noreferrer" class="block truncate font-medium">
				{subdomain}
			</a>
			<div class="truncate text-sm text-muted-foreground">by {author}</div>
		</div>
		{#if onupdate || ondelete}
			<div class="flex shrink-0 gap-2">
				{#if onupdate}
					<button
						onclick={onupdate}
						title="Update"
						class="grid size-8 place-items-center border border-edge hover:border-primary"
					>
						<UpdateIcon />
					</button>
				{/if}
				{#if ondelete}
					<button
						onclick={ondelete}
						title="Delete"
						class={[
							"grid size-8 place-items-center border",
							"border-destructive text-destructive hover:bg-destructive/10",
						]}
					>
						<DeleteIcon />
					</button>
				{/if}
			</div>
		{/if}
	</div>
</div>
