import { cookies } from "next/headers";
import { dicts, type Dict, type Lang } from "./dict";

/** The chosen language, from the same cookie the switch writes. Server-read so
 *  the landing page ships already translated — a page that renders in Uzbek and
 *  then flips to Russian is worse than one that never offered Russian. */
export async function getLang(): Promise<Lang> {
  const v = (await cookies()).get("lang")?.value;
  return v === "ru" || v === "en" ? v : "uz";
}

export async function getT(): Promise<Dict> {
  return dicts[await getLang()];
}
