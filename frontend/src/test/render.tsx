import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";

import AskProvider from "@/components/ui/Ask";
import { LangProvider } from "@/lib/i18n/client";
import { setTillDeviceToken } from "@/lib/api";

/** Bind this browser to a branch, the way the panel's link does.
 *
 *  ⚠️ This is the ordinary installation — a monoblock on a counter with no
 *  staff account signed into it — and every till test starts here unless it is
 *  specifically about the older login-based screen. */
export function bindDevice(token = "device-token") {
  setTillDeviceToken(token);
}

/** Render a till screen in Uzbek, the base language of both dictionaries.
 *
 *  ⚠️ **With the till's own question dialog**, as TillShell mounts it. Without
 *  it `ask()` falls back to `window.confirm`, which jsdom answers "no" — so a
 *  screen that asks before it acts looked, in a test, like a screen that never
 *  acts at all. */
export function renderTill(ui: ReactElement) {
  const user = userEvent.setup();
  const result = render(
    <LangProvider initial="uz">
      <AskProvider look="till">{ui}</AskProvider>
    </LangProvider>,
  );
  return { ...result, user };
}
