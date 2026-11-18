import type { User } from "#lib/auth.svelte.ts";

export interface Site {
	name: string;
	url: string;
	owner: string | null;
	ownerRole: "admin" | "member" | "guest" | "";
	canEdit: boolean;
}

/** Everything needed to draw the first screen */
export interface Boot {
	user: User | null;
	loginEnabled: boolean;
	devLogin: boolean;
	baseDomain: string;
	sites: Site[];
}

/**
 * The server embeds this in the page, so there is normally no request.
 * It falls back to fetching when the page wasn't served by the server (vite dev)
 */
export async function loadBoot(): Promise<Boot> {
	const tag = document.getElementById("boot");
	if (tag?.textContent) {
		const boot = JSON.parse(tag.textContent);
		tag.remove();
		return boot;
	}
	const res = await fetch("/api/boot");
	if (!res.ok) throw new Error(await res.text());
	return res.json();
}

/** Guests pick their own nickname, so they are marked to not pass as real accounts */
export function authorOf(site: Site) {
	if (!site.owner) return undefined;
	return site.ownerRole === "guest" ? `${site.owner} (guest)` : site.owner;
}
