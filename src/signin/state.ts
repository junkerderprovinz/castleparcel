// What the owner's sign-in said about this visitor at the start: whether CastleParcel is still open, and whether
// the owner is signed in in this browser.
import { reactive } from "vue";

export const account = reactive({ known: false, open: true, signedIn: false, passkeys: false, minPassword: 12 });

/** Whether this visitor may not send yet. The signed-in owner always sends. Anyone else needs the upload password
 * when one is set, and once the owner has a password, cannot send without one. */
export const uploadsClosed = (uploadProtected: boolean, token: string) =>
  !account.signedIn && (account.open ? uploadProtected && !token : !(uploadProtected && token));
