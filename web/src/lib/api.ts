/** fetch that sends `json` as the body, and throws the server's message if the request fails */
export async function api(url: string, { json, ...init }: RequestInit & { json?: unknown } = {}) {
	if (json !== undefined) {
		init.headers = { "Content-Type": "application/json" };
		init.body = JSON.stringify(json);
	}
	const res = await fetch(url, init);
	if (!res.ok) throw new Error((await res.text()).trim());
	return res;
}

export const errorText = (e: unknown) => (e instanceof Error ? e.message : String(e));
