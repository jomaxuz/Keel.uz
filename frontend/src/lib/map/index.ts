// Picking the engine, and the one hook every map component starts from.
//
// ⚠️ The three engines are imported statically rather than behind a dynamic
// `import()`. Only one of them ever runs, but they are a few kilobytes each and
// the SDKs themselves — the megabytes — are loaded by whichever engine is
// actually used. A dynamic import here would buy little and add a failure mode
// on a phone with a bad connection, in the middle of picking a delivery address.

import { useEffect, useRef, useState } from "react";

import { normalizeProvider, useMapConfig, type MapProviderId } from "@/lib/map/config";
import type { CreateOptions, MapEngine, MapHandle } from "@/lib/map/engine";
import { googleEngine } from "@/lib/map/google";
import { twogisEngine } from "@/lib/map/twogis";
import { yandexEngine } from "@/lib/map/yandex";

const ENGINES: Record<MapProviderId, MapEngine> = {
  "2gis": twogisEngine,
  yandex: yandexEngine,
  google: googleEngine,
};

export function engineFor(provider: string | undefined): MapEngine {
  return ENGINES[normalizeProvider(provider)];
}

export type MapFailure = null | "webgl" | "load";

function webglAvailable(): boolean {
  try {
    const canvas = document.createElement("canvas");
    return !!(canvas.getContext("webgl") || canvas.getContext("experimental-webgl"));
  } catch {
    return false;
  }
}

/**
 * Creates the map and hands back the handle, with the two ways it fails.
 *
 * Every map on the site went through the same forty lines of this before: probe
 * WebGL, defer a frame so the container is laid out, load the SDK, catch both
 * failures separately, tear down on unmount, and offer a retry. Written five
 * times it was five chances to forget the teardown — and a leaked map keeps its
 * canvas and its listeners alive for as long as the tab does.
 */
export function useMapEngine(opts: {
  container: React.RefObject<HTMLDivElement | null>;
  center: { lat: number; lng: number };
  zoom: number;
  /** Bumped by the caller's retry button. */
  attempt: number;
}) {
  const config = useMapConfig();
  const [handle, setHandle] = useState<MapHandle | null>(null);
  const [failed, setFailed] = useState<MapFailure>(null);
  const handleRef = useRef<MapHandle | null>(null);
  // Read at creation time only: a map is not rebuilt because the caller's centre
  // moved, it is panned (see setCenter). Kept in a ref so it is not a dependency.
  const startRef = useRef<CreateOptions | null>(null);
  startRef.current = { center: opts.center, zoom: opts.zoom, key: config?.key ?? "" };

  const { container, attempt } = opts;
  const key = config?.key ?? null;
  const provider = config?.provider;

  useEffect(() => {
    if (!key || !container.current) return;
    const engine = engineFor(provider);
    if (engine.requiresWebGL && !webglAvailable()) {
      setFailed("webgl");
      return;
    }

    let destroyed = false;
    // One frame, so the container has its size before the SDK measures it.
    const raf = requestAnimationFrame(() => {
      const el = container.current;
      if (!el || !startRef.current) return;
      engine
        .create(el, startRef.current)
        .then((h) => {
          if (destroyed) {
            h.destroy();
            return;
          }
          handleRef.current = h;
          setHandle(h);
          setFailed(null);
        })
        .catch((err) => {
          console.error(`[map:${engine.id}] init failed:`, err);
          // The SDK not arriving and the SDK refusing to draw are different
          // problems with different answers — a blocked script versus a browser
          // that cannot. Told apart here so the message can be.
          setFailed(engine.requiresWebGL ? "webgl" : "load");
        });
    });

    return () => {
      destroyed = true;
      cancelAnimationFrame(raf);
      try {
        handleRef.current?.destroy();
      } catch {
        /* already gone */
      }
      handleRef.current = null;
      setHandle(null);
    };
    // Deliberately not `center`/`zoom`: those move the map, they do not rebuild it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attempt, key, provider]);

  return {
    handle,
    failed,
    /** `null` while the profile is still loading — which must not be shown as
     *  "no key configured", the message that sends owners to a correct setting. */
    hasKey: config == null ? null : config.key !== "",
    provider: config?.provider ?? null,
    setFailed,
  };
}

export type { MapHandle } from "@/lib/map/engine";
export { MAP_PROVIDERS, normalizeProvider } from "@/lib/map/config";
export type { MapProviderId } from "@/lib/map/config";
