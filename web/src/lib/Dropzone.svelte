<script lang="ts">
	let { name, required = false }: { name: string; required?: boolean } = $props();

	let dragging = $state(false);
	let filename = $state<string | null>(null);
</script>

<div
	class={[
		"relative flex min-h-32 flex-col items-center justify-center gap-1",
		"border-2 border-dashed p-6 text-center",
		"has-focus-visible:ring-2 has-focus-visible:ring-primary",
		dragging ? "border-primary bg-primary/10" : "border-edge hover:border-primary",
	]}
>
	<input
		type="file"
		{name}
		{required}
		class="absolute inset-0 size-full cursor-pointer opacity-0"
		ondragenter={() => (dragging = true)}
		ondragleave={() => (dragging = false)}
		ondrop={() => (dragging = false)}
		onchange={(e) => (filename = e.currentTarget.files?.[0]?.name ?? null)}
	/>
	{#if filename}
		<span class="font-medium break-all">{filename}</span>
		<span class="text-sm text-muted-foreground">Click or drop to replace</span>
	{:else}
		<span class="font-medium">Drop a file here</span>
		<span class="text-sm text-muted-foreground">or click to browse</span>
	{/if}
</div>
