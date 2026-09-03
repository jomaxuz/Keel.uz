// What this television plays, and where it keeps it.
//
// ⚠️ **Every file is downloaded onto the set and played from disk.** Not an
// optimisation: a wall-mounted screen loops the same forty-megabyte clip every
// two minutes, all day, on a restaurant's own wifi — the same wifi the till and
// the waiters' phones are on. Streaming it would be a permanent load on the one
// network the business cannot afford to lose, to fetch bytes that never change.
//
// ⚠️ **And it is what makes the screen survive the internet.** A dining room
// keeps working when the connection does not; a television that went black
// every time the router blinked would be unplugged within a week. Once the
// files are on the set, the playlist needs nothing from us to keep going.
//
// ⚠️ **The list is re-read only when the branch says it changed.** The
// heartbeat carries a version number; comparing it costs nothing, and a screen
// that re-downloaded the playlist every minute would ask a question whose
// answer is almost always "the same as last minute".

import { Directory, File, Paths } from "expo-file-system";
import { useCallback, useEffect, useRef, useState } from "react";

import { api } from "@/lib/api";

import { noteServerTime, serverNow } from "./clock";

/** One thing on the wall, with the copy that is actually played. */
export interface PlayItem {
  id: string;
  kind: "image" | "video";
  /** Where it came from, and the name the local copy is keyed by. */
  url: string;
  seconds: number;
  startsAt?: string;
  endsAt?: string;
  /** The local file. ⚠️ This is what is handed to the player, never the URL. */
  localUri: string;
}

/** What is written beside the media, so a cold boot with no network still has
 *  something to play. ⚠️ A file rather than the secure store: a playlist of
 *  sixty rows is far past what Android's keystore is meant to hold, and the
 *  failure there is silent. */
const MANIFEST = "playlist.json";

interface Manifest {
  version: number;
  items: PlayItem[];
}

function contentDir(): Directory {
  // The document directory, not the cache: the system empties the cache when
  // storage runs low, and a television that quietly lost its playlist overnight
  // is a black screen at opening time with nobody to notice why.
  const dir = new Directory(Paths.document, "tv-content");
  if (!dir.exists) dir.create({ intermediates: true, idempotent: true });
  return dir;
}

/** The local name for one URL.
 *
 *  ⚠️ **Derived from the uploaded name, which is already random and unique**,
 *  so a replaced file is a different name and no cache can serve a stale one —
 *  the same property the uploads route relies on for its `immutable` header.
 *  Everything outside a safe alphabet is dropped rather than escaped: this
 *  string becomes a path. */
export function localNameFor(url: string): string {
  const last = url.split("?")[0].split("/").pop() ?? "";
  const safe = last.replace(/[^A-Za-z0-9._-]/g, "");
  return safe && !safe.startsWith(".") ? safe : "";
}

/** Whether a slide may be on screen now.
 *
 *  ⚠️ **The television applies this itself, and that is deliberate.** The
 *  server could trim the list — but a screen that has not reached the internet
 *  since Friday would then still be showing an offer that ended on Saturday,
 *  and an expired promotion on a wall is worse than a blank one: a guest asks
 *  for it at the till. The same rule as models.TVSlidePlayable. */
export function playableNow(item: PlayItem, now: number): boolean {
  if (item.startsAt && now < Date.parse(item.startsAt)) return false;
  if (item.endsAt && now > Date.parse(item.endsAt)) return false;
  return true;
}

function readManifest(): Manifest | null {
  try {
    const file = new File(contentDir(), MANIFEST);
    if (!file.exists) return null;
    const parsed = JSON.parse(file.textSync()) as Manifest;
    if (!Array.isArray(parsed.items)) return null;
    // ⚠️ Only the items whose files are still there. A download interrupted by
    // a power cut leaves a manifest naming a file that does not exist, and the
    // player would sit on it — a still, black television that looks broken and
    // is not.
    return {
      version: parsed.version,
      items: parsed.items.filter((it) => new File(it.localUri).exists),
    };
  } catch {
    return null;
  }
}

function writeManifest(m: Manifest): void {
  try {
    const file = new File(contentDir(), MANIFEST);
    if (file.exists) file.delete();
    file.create();
    file.write(JSON.stringify(m));
  } catch {
    // The screen plays what is in memory this launch and re-downloads on the
    // next one. Nothing a person in the room could do about it either way.
  }
}

/** Remove local files nothing in the list points at any more.
 *
 *  ⚠️ **After the new list is written, never before.** A television with 8 GB
 *  of storage fills up in a season of promotions otherwise — but deleting first
 *  would mean a failed download leaves the set with neither copy, which is the
 *  one outcome the room can see. */
function prune(items: PlayItem[]): void {
  try {
    const keep = new Set(items.map((it) => localNameFor(it.url)));
    keep.add(MANIFEST);
    for (const entry of contentDir().list()) {
      const name = entry.uri.split("/").pop() ?? "";
      if (name && !keep.has(name)) entry.delete();
    }
  } catch {
    // Storage is the set's problem to report, not the room's to look at.
  }
}

/** What this television is playing, kept in step with the branch's playlist.
 *
 *  `contentVersion` is null until the first heartbeat lands — which is exactly
 *  the case a cold boot with no network hits, and why the stored manifest is
 *  read before anything is asked of the server. */
export function useTVPlaylist(contentVersion: number | null) {
  const [items, setItems] = useState<PlayItem[]>([]);
  // What the stored list was built from, so an unchanged version asks nothing.
  const have = useRef<number | null>(null);
  const syncing = useRef(false);
  // ⚠️ **The same fact as `syncing`, in state rather than a ref, and it exists
  // for the wall to read.** A ref cannot re-render, so while a freshly paired
  // screen downloaded its first two videos the room was told "Kontent yo'q —
  // Keel panelida: TV ekranlar → Kontent": an instruction to go and fix
  // something that was working, on a set that was four minutes from playing.
  // Downloading is a normal state and it needs to be a *visible* one.
  const [downloading, setDownloading] = useState(false);

  // ---- The copy on disk, first and without the network ----
  useEffect(() => {
    const stored = readManifest();
    if (stored) {
      have.current = stored.version;
      setItems(stored.items);
    }
  }, []);

  const sync = useCallback(async () => {
    if (syncing.current) return;
    syncing.current = true;
    setDownloading(true);
    try {
      const res = await api.tvPlaylist();
      noteServerTime(res.serverTime);

      const dir = contentDir();
      const next: PlayItem[] = [];
      for (const slide of res.slides) {
        const name = localNameFor(slide.url);
        if (!name) continue;
        const file = new File(dir, name);
        if (!file.exists) {
          // ⚠️ One at a time, in order. A branch that just uploaded six videos
          // would otherwise open six downloads across the restaurant's wifi at
          // once — and the first item, which is the one about to be on screen,
          // would arrive last.
          //
          // ⚠️ **Caught per item, and that is not tidiness.** This `await` used
          // to throw straight out to the silent catch below, which discarded
          // `next` entirely — so one unreachable file lost the whole playlist,
          // *including the clips that had already downloaded*, and the wall
          // said "Kontent yo'q". One bad file should cost one slide.
          try {
            await File.downloadFileAsync(slide.url, file, { idempotent: true });
          } catch {
            continue;
          }
        }
        next.push({
          id: slide.id,
          kind: slide.kind,
          url: slide.url,
          seconds: slide.seconds,
          startsAt: slide.startsAt,
          endsAt: slide.endsAt,
          localUri: file.uri,
        });
      }

      writeManifest({ version: res.version, items: next });
      prune(next);
      // ⚠️ Only when every slide the server listed actually landed. Recording
      // the version after a partial download would make the next heartbeat say
      // "nothing changed" and the missing clips would never be retried — the
      // playlist would stay one item short until somebody edited it.
      if (next.length === res.slides.length) have.current = res.version;
      setItems(next);
    } catch {
      // ⚠️ **Silence, and the old list keeps playing.** A failed download is
      // not a reason to blank a wall: what is already on the set is still the
      // restaurant's content, only one revision behind. The next heartbeat
      // brings the version round again.
    } finally {
      syncing.current = false;
      setDownloading(false);
    }
  }, []);

  useEffect(() => {
    if (contentVersion === null) return;
    if (have.current === contentVersion) return;
    void sync();
  }, [contentVersion, sync]);

  // ---- What may be on screen right now ----
  //
  // ⚠️ **Re-evaluated on a timer, not only when the list changes.** A window
  // closes on its own — at midnight, with nobody watching and quite possibly
  // with no internet — and a screen that only re-checked when the panel changed
  // something would show an expired offer until somebody edited the playlist.
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const id = setInterval(() => setTick((n) => n + 1), 60_000);
    return () => clearInterval(id);
  }, []);

  const now = serverNow();
  void tick;
  return {
    items: items.filter((it) => playableNow(it, now)),
    downloading,
    refresh: sync,
  };
}
