import { goto } from "$app/navigation";
import { createContext, onMount } from "svelte";

export class Editor {
	subdomain = $state("");
	#ready = $state(false);

	#readHash = () => decodeURIComponent(location.hash.slice(1));

	isActive(subdomain: string) {
		return subdomain === this.subdomain;
	}

	syncHash() {
		onMount(() => {
			const load = () => (this.subdomain = this.#readHash());
			load();
			this.#ready = true;
			window.addEventListener("hashchange", load);
			return () => window.removeEventListener("hashchange", load);
		});
		$effect(() => {
			if (!this.#ready || this.subdomain === this.#readHash()) return;
			const hash = this.subdomain ? `#${encodeURIComponent(this.subdomain)}` : "";
			goto(`${location.pathname}${location.search}${hash}`, { shallow: true, replace: true });
		});
	}
}

const [getEditor, setEditorContext] = createContext<Editor>();

export { getEditor };
export const setEditor = () => setEditorContext(new Editor());
