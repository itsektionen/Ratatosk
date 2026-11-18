<script lang="ts">
	import CopyIcon from "~icons/ph/copy-fill";
	import CheckIcon from "~icons/ph/check-bold";
	import CloseIcon from "~icons/ph/x-bold";
	import DeviceIcon from "~icons/ph/devices-fill";
	import InviteIcon from "~icons/ph/user-plus-fill";
	import DevIcon from "~icons/ph/terminal-window-fill";
	import SignInIcon from "~icons/ph/sign-in-fill";
	import SignOutIcon from "~icons/ph/sign-out-fill";
	import { errorText } from "#lib/api.ts";
	import Button from "#lib/Button.svelte";
	import { getAuth, type Link } from "#lib/auth.svelte.ts";
	import { getSites } from "#lib/sites.svelte.ts";

	const auth = getAuth();
	const sites = getSites();

	async function logout() {
		await auth.logout();
		sites.refresh().catch(() => {}); // edit rights are gone
	}

	type Mode = "invites" | "share";
	const durations = [
		{ hours: 1, label: "1h" },
		{ hours: 24, label: "1d" },
		{ hours: 24 * 7, label: "7d" },
		{ hours: 24 * 30, label: "30d" },
	];

	let mode = $state<Mode | null>(null);
	let hours = $state(24 * 7);
	let link = $state<Link | null>(null);
	let error = $state<string | null>(null);
	let copied = $state(false);

	function open(next: Mode) {
		mode = mode === next ? null : next;
		link = error = null;
		copied = false;
	}

	async function create() {
		if (!mode) return;
		error = link = null;
		try {
			link = await auth.createLink(mode, mode === "invites" ? hours : undefined);
		} catch (e) {
			error = errorText(e);
		}
	}

	async function copy() {
		if (!link) return;
		await navigator.clipboard.writeText(link.url);
		copied = true;
	}

	const formatDate = (d: Date) => d.toLocaleString([], { dateStyle: "medium", timeStyle: "short" });
</script>

{#if auth.user === undefined}
	<div class="h-10"></div>
{:else if auth.user}
	<div class="flex flex-col gap-2">
		<div class="flex h-10 items-center gap-2">
			<div class="min-w-0 flex-1 leading-tight">
				<div class="truncate text-sm font-medium">{auth.user.name}</div>
				<div class="truncate text-xs text-muted-foreground capitalize">{auth.user.role}</div>
			</div>
			{#if auth.canInvite}
				<Button
					class="size-8 shrink-0"
					title="Invite guest"
					aria-label="Invite guest"
					onclick={() => open("invites")}
				>
					<InviteIcon />
				</Button>
			{/if}
			{#if auth.canShare}
				<Button
					class="size-8 shrink-0"
					title="Sign in on another device"
					aria-label="Sign in on another device"
					onclick={() => open("share")}
				>
					<DeviceIcon />
				</Button>
			{/if}
			<Button
				variant="danger"
				class="size-8 shrink-0"
				title="Sign out"
				aria-label="Sign out"
				onclick={logout}
			>
				<SignOutIcon />
			</Button>
		</div>
		{#if mode}
			<div
				class={[
					"flex flex-col gap-3 border p-3 text-sm",
					mode === "share" ? "border-destructive" : "border-edge",
				]}
			>
				<div class="flex items-center justify-between gap-2">
					<h2 class="font-medium">
						{mode === "invites" ? "Invite a guest" : "Sign in on another device"}
					</h2>
					<Button
						class="size-6 shrink-0"
						title="Close"
						aria-label="Close"
						onclick={() => (mode = null)}
					>
						<CloseIcon />
					</Button>
				</div>

				{#if mode === "invites"}
					<p class="text-muted-foreground">
						Anyone with this link gets their own guest account, until it expires.
					</p>
					{#if !link}
						<fieldset class="flex flex-col gap-1">
							<legend class="mb-1">Expires after</legend>
							<div class="grid grid-cols-4">
								{#each durations as d (d.hours)}
									<label
										class={[
											"-ml-px cursor-pointer border border-edge py-1 text-center first:ml-0",
											"has-checked:border-foreground has-checked:bg-foreground has-checked:text-background",
											"has-focus-visible:outline-2 has-focus-visible:outline-primary",
											"hover:border-primary has-checked:hover:border-foreground",
										]}
									>
										<input
											type="radio"
											name="expires"
											value={d.hours}
											bind:group={hours}
											class="sr-only"
										/>
										{d.label}
									</label>
								{/each}
							</div>
						</fieldset>
					{/if}
				{:else}
					<p class="text-destructive">
						Anyone with this link is signed in as you. Only open it on your own device.
					</p>
				{/if}

				{#if link}
					<div class="flex flex-col gap-1">
						<div class="flex gap-2">
							<input
								readonly
								value={link.url}
								onfocus={(e) => e.currentTarget.select()}
								class="min-w-0 flex-1 font-mono text-sm"
								aria-label={mode === "invites" ? "Guest invite link" : "Sign-in link for you"}
							/>
							<Button
								class="size-9.5 shrink-0"
								title="Copy link"
								aria-label="Copy link"
								onclick={copy}
							>
								{#if copied}<CheckIcon />{:else}<CopyIcon />{/if}
							</Button>
						</div>
						<span class="text-xs text-muted-foreground">Expires {formatDate(link.expires)}</span>
					</div>
				{:else}
					<Button
						variant={mode === "share" ? "danger" : "default"}
						class="px-3 py-1"
						onclick={create}
					>
						{mode === "invites" ? "Create invite link" : "Create sign-in link"}
					</Button>
				{/if}
				{#if error}
					<p class="text-destructive">{error}</p>
				{/if}
			</div>
		{/if}
	</div>
{:else}
	<div class="flex h-10 items-center gap-2">
		<div class="min-w-0 flex-1 truncate">
			<span class="font-medium">Not signed in</span>
			{#if !auth.loginEnabled && !auth.devLogin}
				<span class="text-xs text-muted-foreground">Ask for an invite link</span>
			{/if}
		</div>
		{#if auth.loginEnabled}
			<Button
				href="/api/auth/login"
				data-sveltekit-reload
				class="size-8 shrink-0"
				title="Sign in"
				aria-label="Sign in"
			>
				<SignInIcon />
			</Button>
		{/if}
		{#if auth.devLogin}
			<form method="POST" action="/api/auth/dev-login">
				<Button type="submit" class="size-8 shrink-0" title="Dev login" aria-label="Dev login">
					<DevIcon />
				</Button>
			</form>
		{/if}
	</div>
{/if}
