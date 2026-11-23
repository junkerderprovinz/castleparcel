import { test } from "node:test";
import assert from "node:assert/strict";
import { account, uploadsClosed } from "../src/signin/state";

function as(open: boolean, signedIn: boolean, check: () => void) {
  Object.assign(account, { open, signedIn });
  try {
    check();
  } finally {
    Object.assign(account, { open: true, signedIn: false });
  }
}

test("an open castle sends as before, with the upload password where one is set", () => {
  as(true, false, () => {
    assert.equal(uploadsClosed(false, ""), false);
    assert.equal(uploadsClosed(true, ""), true);
    assert.equal(uploadsClosed(true, "guest secret"), false);
  });
});

test("once the owner has a password, only guests with the upload password send", () => {
  as(false, false, () => {
    assert.equal(uploadsClosed(false, ""), true);
    assert.equal(uploadsClosed(true, ""), true);
    assert.equal(uploadsClosed(true, "guest secret"), false);
  });
});

test("the signed-in owner always sends", () => {
  as(false, true, () => {
    assert.equal(uploadsClosed(false, ""), false);
    assert.equal(uploadsClosed(true, ""), false);
  });
});
