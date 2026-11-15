<script lang="ts">
	import { page } from "$app/state";
</script>

<main class="grid flex-1 place-items-center p-6 text-center">
	<div class="flex flex-col gap-2">
		<h1 class="text-2xl font-medium">{page.status}</h1>
		<p class="text-muted-foreground">{page.error?.message ?? "Something went wrong"}</p>
		<a href="/">Back</a>
	</div>
</main>
