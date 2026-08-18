import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";

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

/** Render a till screen in Uzbek, the base language of both dictionaries. */
export function renderTill(ui: ReactElement) {
  const user = userEvent.setup();
  const result = render(<LangProvider initial="uz">{ui}</LangProvider>);
  return { ...result, user };
}
