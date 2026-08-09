"use client";

// The strip the restaurant runs itself: this week's discount, a new dish, a holiday.
//
// ⚠️ **Scroll-snap, not a slider library.** The gesture a phone already has is the right
// one, it works before hydration, and it costs no script on the page whose weight was
// worth an afternoon's work. What a library would add is auto-advance and dots; the dots
// are twelve lines below, and the auto-advance is deliberate — see the note on it.
//
// ⚠️ **It advances on its own, and stops the moment somebody touches it.** A promotional
// strip that never moves shows one banner for ever, so the second and third are paid for
// and never seen. A strip that keeps moving while somebody is reading it is the thing
// people complain about — so the timer dies on the first interaction and never comes back.

import { useEffect, useRef, useState } from "react";
import LocaleLink from "@/components/site/LocaleLink";
import { imageUrl } from "@/lib/api";
import { useI18n } from "@/lib/i18n/client";
import { localized } from "@/lib/i18n/site-content";
import type { Banner } from "@/lib/types";

const ADVANCE_MS = 6000;

export default function BannerCarousel({ banners }: { banners: Banner[] }) {
  const { lang } = useI18n();
  const strip = useRef<HTMLDivElement>(null);
  const [index, setIndex] = useState(0);
  // Set by the first touch, drag or dot press, and never unset.
  const touched = useRef(false);

  useEffect(() => {
    if (banners.length < 2) return;
    const id = window.setInterval(() => {
      if (touched.current) {
        window.clearInterval(id);
        return;
      }
      const el = strip.current;
      if (!el) return;
      const next = (Math.round(el.scrollLeft / el.clientWidth) + 1) % banners.length;
      el.scrollTo({ left: next * el.clientWidth, behavior: "smooth" });
    }, ADVANCE_MS);
    return () => window.clearInterval(id);
  }, [banners.length]);

  // Which one is showing, for the dots. Read from the scroll position rather than kept as
  // the source of truth: the guest can swipe, and a counter that disagrees with what is on
  // screen is worse than no counter.
  function onScroll() {
    const el = strip.current;
    if (!el) return;
    setIndex(Math.round(el.scrollLeft / el.clientWidth));
  }

  if (banners.length === 0) return null;

  return (
    <section className="container-page pt-6">
      <div
        ref={strip}
        onScroll={onScroll}
        onPointerDown={() => {
          touched.current = true;
        }}
        className="no-scrollbar flex snap-x snap-mandatory gap-4 overflow-x-auto rounded-3xl"
      >
        {banners.map((b) => {
          const caption = localized(b.title, lang);
          const img = (
            <div className="relative aspect-[16/6] w-full shrink-0 snap-center overflow-hidden rounded-3xl bg-ink/5 sm:aspect-[16/5]">
              {/* Plain <img>: the width is already asked for, and next/image cannot reach
                  our own uploads in this deployment — see SchemaBlocks for the whole
                  story. */}
              <img
                src={imageUrl(b.imageUrl, 1200) ?? ""}
                alt={caption}
                className="absolute inset-0 h-full w-full object-cover"
              />
              {caption && (
                <span className="absolute bottom-0 left-0 right-0 bg-charcoal/55 px-4 py-3 text-sm font-semibold text-white sm:text-base">
                  {caption}
                </span>
              )}
            </div>
          );
          // A banner with no link is a picture, not a broken button.
          return b.link ? (
            <LocaleLink
              key={b.id}
              href={b.link}
              className="w-full shrink-0 snap-center sm:w-[calc(100%-0px)]"
            >
              {img}
            </LocaleLink>
          ) : (
            <div key={b.id} className="w-full shrink-0 snap-center">
              {img}
            </div>
          );
        })}
      </div>

      {banners.length > 1 && (
        <div className="mt-3 flex justify-center gap-1.5">
          {banners.map((b, i) => (
            <button
              key={b.id}
              type="button"
              aria-label={`${i + 1}`}
              onClick={() => {
                touched.current = true;
                strip.current?.scrollTo({
                  left: i * (strip.current?.clientWidth ?? 0),
                  behavior: "smooth",
                });
              }}
              className={`h-1.5 rounded-full transition-all ${
                i === index ? "w-6 bg-brand" : "w-1.5 bg-ink/20"
              }`}
            />
          ))}
        </div>
      )}
    </section>
  );
}
