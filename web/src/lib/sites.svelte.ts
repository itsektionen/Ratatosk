import type { Site } from "#lib/boot.ts";
import { createContext } from "svelte";

export class Sites {
	list = $state<Site[]>([]);
	/** Sites live at `<name>.<baseDomain>` */
	baseDomain = $state("");
	ready = $state(false);

	async refresh() {
		const res = await fetch("/api/sites");
		if (!res.ok) throw new Error(await res.text());
		this.list = await res.json();
	}

	async rename(name: string, to: string) {
		const res = await fetch(`/api/sites/${encodeURIComponent(name)}`, {
			method: "PATCH",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ name: to }),
		});
		if (!res.ok) throw new Error(await res.text());
		await this.refresh(); // the URL changed too
	}

	async remove(name: string) {
		const res = await fetch(`/api/sites/${encodeURIComponent(name)}`, { method: "DELETE" });
		if (!res.ok) throw new Error(await res.text());
		this.list = this.list.filter((s) => s.name !== name);
	}
}

const [getSites, setSitesContext] = createContext<Sites>();

export { getSites };
export const setSites = () => setSitesContext(new Sites());
