import { test } from "node:test";
import assert from "node:assert/strict";
import { createAgain } from "../src/fortress/again";

const notFresh = Object.assign(new Error("signInAgain"), { code: "signInAgain", askPassword: true });

/** A change that wants a fresh sign-in the first time and goes through after it. */
function change() {
  let calls = 0;
  return {
    get calls() {
      return calls;
    },
    run: async () => {
      calls++;
      if (calls === 1) throw notFresh;
      return "the phrase";
    },
  };
}

const tick = () => new Promise((resolve) => setTimeout(resolve, 0));

test("a change that wants a fresh sign-in asks for the password and runs again", async () => {
  const tried: string[] = [];
  const again = createAgain(async (password) => {
    tried.push(password);
    return password === "right" ? { ok: true, value: {} } : { ok: false, status: 403, error: "wrongPassword" };
  });
  again.reauth.subscribe(() => {});
  const c = change();
  const done = again.fresh(c.run);
  await tick();
  assert.equal(again.reauth.state().asking, true);

  again.reauth.set("wrong");
  await again.reauth.submit();
  assert.equal(again.reauth.state().error, "wrongPassword");
  assert.equal(again.reauth.state().refused, 1);
  assert.equal(again.reauth.state().asking, true);

  again.reauth.set("right");
  await again.reauth.submit();
  assert.equal(await done, "the phrase");
  assert.deepEqual(tried, ["wrong", "right"]);
  assert.equal(c.calls, 2);
  assert.equal(again.reauth.state().asking, false);
  assert.equal(again.reauth.state().password, "");
});

test("closing the question keeps the change from running", async () => {
  const again = createAgain(async () => ({ ok: true, value: {} }));
  again.reauth.subscribe(() => {});
  const c = change();
  const done = again.fresh(c.run);
  await tick();
  again.reauth.cancel();
  await assert.rejects(done, (e: { code?: string }) => e.code === "cancelled");
  assert.equal(c.calls, 1);
});

test("without a window to ask in, or where the sign-in asks nothing, the refusal stands", async () => {
  const quiet = createAgain(async () => ({ ok: true, value: {} }));
  await assert.rejects(quiet.fresh(change().run), (e) => e === notFresh);

  const noQuestion = createAgain(async () => ({ ok: true, value: {} }));
  noQuestion.reauth.subscribe(() => {});
  const off = Object.assign(new Error("signInAgain"), { code: "signInAgain", askPassword: false });
  await assert.rejects(
    noQuestion.fresh(async () => {
      throw off;
    }),
    (e) => e === off,
  );
  assert.equal(noQuestion.reauth.state().asking, false);
});
