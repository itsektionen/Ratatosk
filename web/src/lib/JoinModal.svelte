<script lang="ts">
	import { api, errorText } from "#lib/api.ts";
	import Button from "#lib/Button.svelte";
	import { onMount } from "svelte";

	// A guest invite link lands here as /?invite=<token>. The guest has to pick
	// a nickname before the server gives them a session
	let token = $state<string | null>(null);
	let dialog = $state<HTMLDialogElement>();
	let name = $state("");
	let busy = $state(false);
	let error = $state<string | null>(null);

	onMount(() => {
		token = new URLSearchParams(location.search).get("invite");
	});
	$effect(() => {
		if (token && dialog && !dialog.open) dialog.showModal();
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		try {
			await api("/api/auth/join", { method: "POST", json: { token, name } });
			location.replace("/"); // reload as the new guest, without the token in the URL
		} catch (e) {
			error = errorText(e);
			busy = false;
		}
	}
</script>

{#if token}
	<dialog
		bind:this={dialog}
		oncancel={(e) => e.preventDefault()}
		class={[
			"m-auto w-[min(100%-2rem,24rem)] border border-edge bg-background p-6 text-foreground",
			"backdrop:bg-background/80",
		]}
	>
		<form onsubmit={submit} class="flex flex-col gap-4">
			<h2 class="text-lg font-medium">Pick a nickname</h2>
			<p class="text-sm text-muted-foreground">
				Others see it next to your sites, marked as a guest.
			</p>
			<input
				bind:value={name}
				required
				maxlength="32"
				autocomplete="off"
				placeholder="Nickname"
				aria-label="Nickname"
			/>
			<Button type="submit" variant="primary" disabled={busy || !name.trim()} class="px-4 py-2">
				Continue
			</Button>
			{#if error}
				<p class="text-sm text-destructive">{error}</p>
			{/if}
		</form>
	</dialog>
{/if}
