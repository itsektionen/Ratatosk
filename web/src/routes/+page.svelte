<script lang="ts">
	import AccountPanel from "#lib/AccountPanel.svelte";
	import { getAuth } from "#lib/auth.svelte.ts";
	import Button from "#lib/Button.svelte";
	import Dropzone from "#lib/Dropzone.svelte";
	import JoinModal from "#lib/JoinModal.svelte";
	import SiteCard from "#lib/SiteCard.svelte";
	import { setEditor } from "#lib/editor.svelte.ts";

	// TODO: Dynamic domain name

	const auth = getAuth();
	const editor = setEditor();
	editor.syncHash();

	let subdomainInput: HTMLInputElement | undefined = $state();

	function edit(subdomain: string) {
		editor.subdomain = subdomain;
		subdomainInput?.focus();
	}
</script>

<main class="grid flex-1 grid-rows-[auto_1fr] md:grid-cols-[auto_minmax(0,1fr)] md:grid-rows-1">
	<aside
		class={[
			"divide-y divide-border max-md:border-b",
			"md:sticky md:top-0 md:h-dvh md:w-90 md:overflow-y-auto md:overscroll-contain md:border-r",
		]}
	>
		<div class="p-4">
			<AccountPanel />
		</div>
		<form
			method="POST"
			action="/api/upload"
			enctype="multipart/form-data"
			class="flex flex-col gap-4 p-4"
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
			<Button type="submit" variant="primary" disabled={!auth.user} class="px-4 py-2 font-medium">
				Upload
			</Button>
		</form>
	</aside>

	<section
		class={[
			"grid grid-cols-[repeat(auto-fill,minmax(min(100%,--spacing(80)),1fr))]",
			"content-start gap-4 p-4",
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

<JoinModal />
