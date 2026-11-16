import { createContext } from "svelte";

export interface User {
	id: number;
	name: string;
	role: "admin" | "member" | "guest";
}

export interface Link {
	url: string;
	expires: Date;
}

export class Auth {
	/** undefined until the first response arrives, null when signed out */
	user = $state<User | null | undefined>();
	loginEnabled = $state(false);
	devLogin = $state(false);

	canInvite = $derived(this.user?.role === "admin" || this.user?.role === "member");
	/** Guests only, everyone else signs in through SSO */
	canShare = $derived(this.user?.role === "guest");

	async refresh() {
		try {
			const res = await fetch("/api/auth/me");
			if (!res.ok) throw new Error(res.statusText);
			const me = await res.json();
			this.user = me.user;
			this.loginEnabled = me.loginEnabled;
			this.devLogin = me.devLogin;
		} catch {
			this.user = null;
		}
	}

	async logout() {
		await fetch("/api/auth/logout", { method: "POST" });
		this.user = null;
	}

	/** "invites" is for a new guest, "share" signs another device in as you */
	async createLink(kind: "invites" | "share", hours?: number): Promise<Link> {
		const res = await fetch(`/api/auth/${kind}`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(hours ? { hours } : {}),
		});
		if (!res.ok) throw new Error(await res.text());
		const { url, expires } = await res.json();
		return { url, expires: new Date(expires) };
	}
}

const [getAuth, setAuthContext] = createContext<Auth>();

export { getAuth };
export const setAuth = () => setAuthContext(new Auth());
