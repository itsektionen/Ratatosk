import { api } from "#lib/api.ts";
import type { Site } from "#lib/boot.ts";
import { createContext } from "svelte";

export class Sites {
	list = $state<Site[]>([]);
	/** Sites live at `<name>.<baseDomain>` */
	baseDomain = $state("");
	/** Bumped when a site's files are replaced, so its preview reloads */
	revisions = $state<Record<string, number>>({});

	async refresh() {
		this.list = await (await api("/api/sites")).json();
	}

	async upload(form: FormData) {
		await api("/api/upload", { method: "POST", body: form });
		// Same normalization as the server
		const name = String(form.get("subdomain")).trim().toLowerCase();
		this.revisions[name] = (this.revisions[name] ?? 0) + 1;
		await this.refresh();
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
