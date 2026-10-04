"use client";

import { useState } from "react";

// The Pitch Day demo video.
//
// ⚠️ **Three states, and the third is honest.** A YouTube id loads YouTube only
// when somebody presses play (an iframe on first paint is ~1 MB of script for a
// video most visitors scroll past); an MP4 is a native player with
// `preload="none"`; and nothing configured is a placeholder that *says* the
// video is coming. A player-looking box that does nothing on click would be the
// one fake thing on a page whose whole argument is that nothing is fake.
//
// The source is set in `PITCH_CONFIG.video` (src/lib/pitch.ts).

export default function PitchVideo({
  youtubeId,
  mp4,
  poster,
  duration,
  labels,
}: {
  youtubeId: string;
  mp4: string;
  poster: string;
  duration: string;
  labels: { play: string; placeholderTitle: string; placeholderLead: string; title: string };
}) {
  const [playing, setPlaying] = useState(false);
  const frame =
    "relative aspect-video w-full overflow-hidden rounded-2xl border border-line-strong bg-hull-950 shadow-xl shadow-hull-950/10";

  if (youtubeId) {
    return (
      <div className={frame}>
        {playing ? (
          <iframe
            className="absolute inset-0 h-full w-full"
            src={`https://www.youtube-nocookie.com/embed/${encodeURIComponent(youtubeId)}?autoplay=1&rel=0`}
            title={labels.title}
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
            allowFullScreen
          />
        ) : (
          <button
            type="button"
            onClick={() => setPlaying(true)}
            className="group absolute inset-0 h-full w-full focus:outline-none focus-visible:ring-4 focus-visible:ring-signal-500/60"
            aria-label={labels.play}
          >
            <img
              src={poster}
              alt=""
              loading="lazy"
              decoding="async"
              className="absolute inset-0 h-full w-full object-cover opacity-80 transition group-hover:opacity-90"
            />
            <PlayBadge duration={duration} />
          </button>
        )}
      </div>
    );
  }

  if (mp4) {
    return (
      <div className={frame}>
        <video
          className="absolute inset-0 h-full w-full"
          src={mp4}
          poster={poster}
          controls
          preload="none"
          playsInline
        >
          <track kind="captions" />
        </video>
      </div>
    );
  }

  return (
    <div className={`${frame} grid place-items-center`} role="img" aria-label={labels.placeholderTitle}>
      <img
        src={poster}
        alt=""
        loading="lazy"
        decoding="async"
        className="absolute inset-0 h-full w-full object-cover opacity-25 blur-[2px]"
      />
      <div className="relative mx-6 max-w-md rounded-2xl border border-white/15 bg-hull-950/80 px-6 py-5 text-center backdrop-blur-sm">
        <span className="mx-auto grid h-12 w-12 place-items-center rounded-full border border-dashed border-signal-400/70 text-signal-400">
          <svg viewBox="0 0 24 24" className="ml-0.5 h-5 w-5" fill="currentColor" aria-hidden>
            <path d="M8 5.5v13l10.5-6.5L8 5.5Z" />
          </svg>
        </span>
        <p className="mt-3 font-display text-lg font-semibold text-white">{labels.placeholderTitle}</p>
        <p className="mt-1 text-sm text-white/70">{labels.placeholderLead}</p>
      </div>
    </div>
  );
}

function PlayBadge({ duration }: { duration: string }) {
  return (
    <span className="absolute inset-0 grid place-items-center">
      <span className="flex items-center gap-3 rounded-full bg-signal-500 py-3 pl-4 pr-5 font-semibold text-hull-950 shadow-lg transition group-hover:scale-[1.03] motion-reduce:transition-none">
        <svg viewBox="0 0 24 24" className="h-5 w-5" fill="currentColor" aria-hidden>
          <path d="M8 5.5v13l10.5-6.5L8 5.5Z" />
        </svg>
        {duration || "Demo"}
      </span>
    </span>
  );
}
