// The fortress routes of CastleParcel's server. While CastleParcel is open and has an upload password, that password
// guards them too, so every call carries it.
import { authHeaders } from "@/src/upload";
import type { FortressInfo, MemberPermission, MemberStatus, RemovedMember } from "./view";

/** A refusal of the server, by the code its body names, or "offline" when it did not answer. askPassword says that a
 * fresh sign-in with the password lets the change through. */
export class FortressError extends Error {
  constructor(
    readonly code: string,
    readonly askPassword = false,
  ) {
    super(code);
  }
}

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { ...authHeaders() };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  let res: Response;
  try {
    res = await fetch(path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  } catch {
    throw new FortressError("offline");
  }
  const got = await res.json().catch(() => ({}));
  if (!res.ok) throw new FortressError(typeof got.error === "string" ? got.error : "internal", got.askPassword === true);
  return got as T;
}

export type ShownPhrase = { words: string[]; code: string };

export const fortressApi = {
  read: () => call<FortressInfo>("GET", "/api/fortress"),
  join: (words: string[]) => call<FortressInfo>("POST", "/api/fortress/join", { phrase: words.join(" ") }),
  leave: () => call<FortressInfo>("DELETE", "/api/fortress"),
  remove: (id: string) => call<FortressInfo>("DELETE", `/api/fortress/members/${encodeURIComponent(id)}`),
  permission: (p: MemberPermission) => call<FortressInfo>("PUT", "/api/fortress/permissions", p),
  showPhrase: () => call<ShownPhrase>("POST", "/api/fortress/phrase/show"),
  changePhrase: () => call<ShownPhrase & { left: RemovedMember[] }>("POST", "/api/fortress/phrase"),
  open: (id: string) => call<{ path: string }>("POST", `/api/fortress/open/${encodeURIComponent(id)}`),
  status: (id: string, lang: string) =>
    call<MemberStatus>("GET", `/api/fortress/members/${encodeURIComponent(id)}/status?lang=${encodeURIComponent(lang)}`),
};
