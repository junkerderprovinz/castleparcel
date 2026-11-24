// Showing or changing the phrase lets any device into the fortress, so the server wants a fresh sign-in for it, as
// the kit wants one for its own lasting changes. The question works like the kit's: GlimStone's window asks for the
// password, the sign-in confirms it, and the change runs once more.
import type { SignInReauth, SignInReauthState, SignInResult } from "@glimstone/core";

type Refusal = { code?: unknown; askPassword?: unknown };

const wantsPassword = (e: unknown) => {
  const r = (e ?? {}) as Refusal;
  return r.code === "signInAgain" && r.askPassword === true;
};

export function createAgain(confirm: (password: string) => Promise<SignInResult<unknown>>) {
  let state: SignInReauthState = { asking: false, password: "", busy: false, error: null, refused: 0 };
  const listeners = new Set<() => void>();
  let waiting: ((confirmed: boolean) => void) | null = null;
  const update = (change: Partial<SignInReauthState>) => {
    state = { ...state, ...change };
    listeners.forEach((fn) => fn());
  };
  const settle = (confirmed: boolean) => {
    const resume = waiting;
    waiting = null;
    update({ asking: false, password: "", busy: false, error: null });
    resume?.(confirmed);
  };

  const reauth: SignInReauth = {
    state: () => state,
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    set: (password) => update({ password, error: null }),
    async submit() {
      if (!state.asking || state.busy || state.password === "") return;
      update({ busy: true, error: null });
      const r = await confirm(state.password);
      if (r.ok) settle(true);
      else update({ busy: false, error: r.error, refused: state.refused + 1 });
    },
    cancel: () => settle(false),
  };

  const ask = () =>
    new Promise<boolean>((resolve) => {
      waiting = resolve;
      update({ asking: true, password: "", error: null });
    });

  /** Runs call, and once more after the password where the server wants a fresh sign-in first. */
  async function fresh<T>(call: () => Promise<T>): Promise<T> {
    try {
      return await call();
    } catch (e) {
      if (!wantsPassword(e) || listeners.size === 0 || waiting) throw e;
      if (!(await ask())) throw Object.assign(new Error("cancelled"), { code: "cancelled" });
      return call();
    }
  }

  return { reauth, fresh };
}
