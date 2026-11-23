// The kit's sign-in as CastleParcel talks to it. While CastleParcel is open and has an upload password, that
// password guards the sign-in's changes too, so every call carries it.
import { createSignInApi } from "@glimstone/core";
import { authHeaders } from "@/src/upload";
import { account } from "./state";

export const signIn = createSignInApi({
  fetch: (input, init) => {
    const headers = new Headers(init?.headers);
    for (const [name, value] of Object.entries(authHeaders())) headers.set(name, value);
    return fetch(input, { ...init, headers });
  },
});

/** Asks the server where the sign-in stands. A server that does not answer leaves CastleParcel as open as before. */
export async function loadAccount() {
  const r = await signIn.options();
  if (r.ok) Object.assign(account, { open: r.value.open, signedIn: r.value.signedIn, passkeys: r.value.passkeys, minPassword: r.value.minPassword });
  account.known = true;
}

// The note the page shows after the reload that follows a sign-in or sign-out.
const NOTE = "cp-signin-note";
export function leaveNote(text: string) {
  try {
    sessionStorage.setItem(NOTE, text);
  } catch {
    // The page then simply says nothing.
  }
}
export function takeNote(): string {
  try {
    const text = sessionStorage.getItem(NOTE) ?? "";
    sessionStorage.removeItem(NOTE);
    return text;
  } catch {
    return "";
  }
}

export async function signOut(bye: string) {
  await signIn.logout();
  leaveNote(bye);
  location.reload();
}
