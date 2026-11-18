<script lang="ts">
	import type { HTMLAnchorAttributes, HTMLButtonAttributes } from "svelte/elements";

	let {
		variant = "default",
		class: className,
		children,
		...rest
	}: { variant?: "default" | "primary" | "danger" } & (
		(HTMLAnchorAttributes & { href: string }) | (HTMLButtonAttributes & { href?: undefined })
	) = $props();
</script>

{#if rest.href !== undefined}
	<a {...rest} class={["btn", variant, className]}>{@render children?.()}</a>
{:else}
	<button type="button" {...rest} class={["btn", variant, className]}>
		{@render children?.()}
	</button>
{/if}

<style>
	.btn {
		--line: var(--color-edge);
		--ink: var(--color-foreground);
		--base: transparent;
		--fill: var(--color-foreground);
		--on-fill: var(--color-background);

		display: grid;
		place-items: center;
		cursor: pointer;
		border: 1px solid var(--line);
		background: var(--base);
		color: var(--ink);
		text-decoration: none;
	}
	.primary {
		--line: var(--color-primary);
		--ink: var(--color-primary-foreground);
		--base: var(--color-primary);
	}
	.danger {
		--line: var(--color-destructive);
		--ink: var(--color-destructive);
		--fill: var(--color-destructive);
	}

	.btn:hover:not(:disabled, [aria-disabled="true"]),
	.btn:focus-visible {
		background: var(--fill);
		color: var(--on-fill);
	}

	.btn:focus-visible {
		outline: 2px solid var(--color-primary);
		outline-offset: 2px;
	}
	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
