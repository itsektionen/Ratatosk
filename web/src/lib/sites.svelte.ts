import { api } from "#lib/api.ts";
import type { Site } from "#lib/boot.ts";
import { createContext } from "svelte";

export class Sites {
	list = $state<Site[]>([]);
	/** Sites live at `<name>.<baseDomain>` */
	baseDomain = $state("");

	async refresh() {
		this.list = await (await api("/api/sites")).json();
	}

	async rename(name: string, to: string) {
		await api(`/api/sites/${encodeURIComponent(name)}`, { method: "PATCH", json: { name: to } });
		await this.refresh(); // the URL changed too
	}

	async remove(name: string) {
		await api(`/api/sites/${encodeURIComponent(name)}`, { method: "DELETE" });
		this.list = this.list.filter((s) => s.name !== name);
	}
}

const [getSites, setSitesContext] = createContext<Sites>();

export { getSites };
export const setSites = () => setSitesContext(new Sites());
