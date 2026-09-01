import { readFileSync } from "node:fs";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

/**
 * ⚠️ **A phone app that registers as the wrong app receives nothing, and says
 * nothing.**
 *
 * Both Android apps sign in as staff, and the server picks the Android
 * notification channel from the `app` field sent with the token: "kitchen" for
 * Keel Waiter, "team" for Keel Team. Android drops a notification addressed to
 * a channel the phone never created — no error on the device, none in the send
 * result, and the registration itself succeeded. So the settings screen shows
 * green, the server reports the message as delivered, and the employee is
 * simply never told anything.
 *
 * That is what Keel Team did for its whole first release: the shared
 * `staffRegisterPush` did not take the field, so every Team phone was stored as
 * a waiter's and every message went to a channel that did not exist there.
 *
 * The bug is invisible in both files on their own — each looks correct — so it
 * is sealed here, where the two are read together: whatever channel an app
 * creates is the app it must register as.
 */
const APPS = ["waiter", "team"] as const;

function read(app: string, file: string): string {
  return readFileSync(join(process.cwd(), "..", "mobile", app, "src", file), "utf8");
}

describe("phone apps register as themselves", () => {
  it.each(APPS)("%s sends its own app name with the token", (app) => {
    const src = read(app, "push.ts");
    expect(src).toContain(`staffRegisterPush(value, Platform.OS, lang, "${app}")`);
  });

  it.each(APPS)("%s creates the channel the server will address", (app) => {
    const src = read(app, "push.ts");
    // The server maps "team" to the team channel and everything else to the
    // kitchen one; the app has to create exactly that channel.
    const channel = app === "team" ? "team" : "kitchen";
    expect(src).toContain(`const CHANNEL = "${channel}"`);
  });
});
