<script lang="ts">
	import AccountPanel from "#lib/AccountPanel.svelte";
	import { api, errorText } from "#lib/api.ts";
	import { getAuth } from "#lib/auth.svelte.ts";
	import { authorOf } from "#lib/boot.ts";
	import Button from "#lib/Button.svelte";
	import Dropzone from "#lib/Dropzone.svelte";
	import JoinModal from "#lib/JoinModal.svelte";
	import SiteCard from "#lib/SiteCard.svelte";
	import { setEditor } from "#lib/editor.svelte.ts";
	import { getSites } from "#lib/sites.svelte.ts";

	// A page or an archive of the site, the server also checks
	const SITE_FILES = ".html,.htm,.zip,.tar,.tar.gz,.tgz";

	const auth = getAuth();
	const sites = getSites();
	const editor = setEditor();
	editor.syncHash();

	let subdomainInput: HTMLInputElement | undefined = $state();

	function edit(subdomain: string) {
		editor.subdomain = subdomain;
		subdomainInput?.focus();
	}

	let uploading = $state(false);
	let message = $state<{ error: boolean; text: string } | null>(null);

	const fail = (e: unknown) => ({ error: true, text: errorText(e) });

	async function upload(e: SubmitEvent & { currentTarget: HTMLFormElement }) {
		e.preventDefault();
		uploading = true;
		message = null;
		try {
			await api("/api/upload", { method: "POST", body: new FormData(e.currentTarget) });
			await sites.refresh();
			message = { error: false, text: "Uploaded" };
		} catch (e) {
			message = fail(e);
		} finally {
			uploading = false;
		}
	}

	async function rename(name: string) {
		const to = prompt(`Rename ${name} to`, name)?.trim().toLowerCase();
		if (!to || to === name) return;
		try {
			await sites.rename(name, to);
			if (editor.subdomain === name) editor.subdomain = to;
		} catch (e) {
			message = fail(e);
		}
	}

	async function remove(name: string) {
		if (!confirm(`Delete ${name}?`)) return;
		try {
			await sites.remove(name);
		} catch (e) {
			message = fail(e);
		}
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
		<form onsubmit={upload} class="flex flex-col gap-4 p-4">
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
				<span class="text-muted-foreground select-none">.{sites.baseDomain}</span>
			</label>
			<Dropzone name="file" accept={SITE_FILES} />
			<Button
				type="submit"
				variant="primary"
				disabled={!auth.user || uploading}
				class="px-4 py-2 font-medium"
			>
				Upload
			</Button>
			{#if message}
				<p class={["text-sm", message.error ? "text-destructive" : "text-muted-foreground"]}>
					{message.text}
				</p>
			{/if}
		</form>
	</aside>

	<section
		class={[
			"grid grid-cols-[repeat(auto-fill,minmax(min(100%,--spacing(80)),1fr))]",
			"content-start gap-4 p-4",
		]}
	>
		{#each sites.list as site (site.name)}
			<SiteCard
				subdomain={site.name}
				url={site.url}
				author={authorOf(site)}
				onupdate={site.canEdit ? () => edit(site.name) : undefined}
				onrename={site.canEdit ? () => rename(site.name) : undefined}
				ondelete={site.canEdit ? () => remove(site.name) : undefined}
			/>
		{/each}
	</section>
</main>

<JoinModal />
