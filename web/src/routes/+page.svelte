<script lang="ts">
	import Dropzone from "#lib/Dropzone.svelte";
	import SiteCard from "#lib/SiteCard.svelte";
	import { setEditor } from "#lib/editor.svelte.ts";

	// TODO: Dynamic domain name

	const editor = setEditor();
	editor.syncHash();

	let subdomainInput: HTMLInputElement | undefined = $state();

	function edit(subdomain: string) {
		editor.subdomain = subdomain;
		subdomainInput?.focus();
	}
</script>

<main class="grid flex-1 grid-rows-[auto_1fr] lg:grid-cols-[auto_minmax(0,1fr)] lg:grid-rows-1">
	<aside
		class={[
			"border-border p-6 max-lg:border-b max-lg:pb-3",
			"lg:sticky lg:top-0 lg:h-dvh lg:w-90 lg:overflow-y-auto lg:overscroll-contain lg:border-r lg:pr-3",
		]}
	>
		<form
			method="POST"
			action="/api/sites"
			enctype="multipart/form-data"
			class="flex flex-col gap-4"
		>
			<label
				class={[
					"flex items-center border border-edge px-3",
					"focus-within:border-primary focus-within:ring-1 focus-within:ring-primary",
				]}
			>
				<input
					type="text"
					name="subdomain"
					bind:value={editor.subdomain}
					bind:this={subdomainInput}
					placeholder="example"
					required
					pattern="[a-z0-9]([a-z0-9\-]*[a-z0-9])?"
					autocomplete="off"
					spellcheck="false"
					class="min-w-0 flex-1 border-0 px-0 text-right focus:ring-0"
				/>
				<span class="text-muted-foreground select-none">.tvs.kth.it</span>
			</label>
			<Dropzone name="file" />
			<button type="submit">Upload</button>
		</form>
	</aside>

	<section
		class={[
			"grid grid-cols-[repeat(auto-fill,minmax(min(100%,--spacing(80)),1fr))]",
			"content-start gap-4 p-6 max-lg:pt-3 lg:pl-3",
		]}
	>
		{#each Array(10) as _, i (i)}
			<SiteCard
				subdomain="example"
				url="https://example.com"
				author="Someone"
				onupdate={() => edit("example")}
				ondelete={() => {}}
			/>
		{/each}
	</section>
</main>
