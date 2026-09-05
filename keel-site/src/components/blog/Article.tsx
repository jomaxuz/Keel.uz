"use client";

// An article, drawn from the text it was written in.
//
// ⚠️ **React elements, never `dangerouslySetInnerHTML`.** The author is one of
// us, so this is not a fear of strangers — it is that a page assembled from
// stored HTML can break the layout of everything around it, and one paste out
// of a word processor brings a font, a colour and a table with it. Building
// elements means an article can only ever be the things this file draws.
//
// ⚠️ **A small language, on purpose.** Headings, paragraphs, lists, quotes,
// links, pictures and a video player. Anything more is something to learn
// before writing, and the point of a blog is that somebody writes in it this
// week.

import { useMemo } from "react";

type Block =
  | { kind: "h2" | "h3" | "p" | "quote"; text: string }
  | { kind: "ul"; items: string[] }
  | { kind: "img"; src: string; alt: string }
  | { kind: "video"; id: string };

/** A YouTube link, in the shapes people actually paste.
 *
 *  ⚠️ **The id is extracted rather than the URL embedded.** A link copied from
 *  the app carries a playlist, a start time and a tracking parameter, and
 *  handing all of that to an iframe is handing a page of ours to whatever those
 *  mean this month. */
const YT =
  /^(?:https?:\/\/)?(?:www\.|m\.)?(?:youtube\.com\/(?:watch\?v=|embed\/|shorts\/)|youtu\.be\/)([A-Za-z0-9_-]{6,20})/;

const IMG = /^!\[([^\]]*)\]\(([^)\s]+)\)$/;

export function parse(body: string): Block[] {
  const out: Block[] = [];
  // ⚠️ Windows line endings included: the text arrives from a textarea on
  // somebody's laptop, and a "\r" left in place turns every heading into a
  // paragraph that merely looks like one.
  const lines = body.replace(/\r/g, "").split("\n");
  let para: string[] = [];
  let list: string[] = [];

  const flushPara = () => {
    if (para.length) out.push({ kind: "p", text: para.join(" ") });
    para = [];
  };
  const flushList = () => {
    if (list.length) out.push({ kind: "ul", items: list });
    list = [];
  };
  const flush = () => {
    flushPara();
    flushList();
  };

  for (const raw of lines) {
    const line = raw.trim();
    if (!line) {
      flush();
      continue;
    }
    const yt = YT.exec(line);
    if (yt) {
      flush();
      out.push({ kind: "video", id: yt[1] });
      continue;
    }
    const img = IMG.exec(line);
    if (img) {
      flush();
      out.push({ kind: "img", alt: img[1], src: img[2] });
      continue;
    }
    if (line.startsWith("### ")) {
      flush();
      out.push({ kind: "h3", text: line.slice(4) });
      continue;
    }
    if (line.startsWith("## ")) {
      flush();
      out.push({ kind: "h2", text: line.slice(3) });
      continue;
    }
    if (line.startsWith("> ")) {
      flush();
      out.push({ kind: "quote", text: line.slice(2) });
      continue;
    }
    if (line.startsWith("- ")) {
      flushPara();
      list.push(line.slice(2));
      continue;
    }
    flushList();
    para.push(line);
  }
  flush();
  return out;
}

/** Bold, italic and links inside a line.
 *
 *  ⚠️ **A link is checked before it is drawn.** `javascript:` in an href is the
 *  one way a piece of stored text can still run something, and an editor
 *  pasting a link somebody sent them is exactly how it would arrive. */
function inline(text: string, key: string) {
  const parts: React.ReactNode[] = [];
  const re = /(\*\*[^*]+\*\*)|(\*[^*]+\*)|(\[[^\]]+\]\([^)\s]+\))/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let n = 0;
  while ((m = re.exec(text))) {
    if (m.index > last) parts.push(text.slice(last, m.index));
    const tok = m[0];
    n += 1;
    if (tok.startsWith("**")) {
      parts.push(<strong key={`${key}-b${n}`}>{tok.slice(2, -2)}</strong>);
    } else if (tok.startsWith("*")) {
      parts.push(<em key={`${key}-i${n}`}>{tok.slice(1, -1)}</em>);
    } else {
      const cut = tok.indexOf("](");
      const label = tok.slice(1, cut);
      const href = tok.slice(cut + 2, -1);
      const safe = /^(https?:\/\/|\/|mailto:)/i.test(href);
      parts.push(
        safe ? (
          <a
            key={`${key}-a${n}`}
            href={href}
            className="underline decoration-signal-500/50 underline-offset-2 hover:text-signal-600"
            {...(href.startsWith("http")
              ? { target: "_blank", rel: "noopener noreferrer" }
              : {})}
          >
            {label}
          </a>
        ) : (
          label
        ),
      );
    }
    last = m.index + tok.length;
  }
  if (last < text.length) parts.push(text.slice(last));
  return parts;
}

export default function Article({ body }: { body: string }) {
  const blocks = useMemo(() => parse(body), [body]);
  return (
    <div className="space-y-5 text-[17px] leading-relaxed text-ink-soft">
      {blocks.map((b, i) => {
        const key = `b${i}`;
        switch (b.kind) {
          case "h2":
            return (
              <h2 key={key} className="h-display pt-3 text-2xl text-ink">
                {b.text}
              </h2>
            );
          case "h3":
            return (
              <h3 key={key} className="pt-2 font-display text-lg font-semibold text-ink">
                {b.text}
              </h3>
            );
          case "quote":
            return (
              <blockquote
                key={key}
                className="border-l-2 border-signal-500/60 pl-4 italic text-ink-muted"
              >
                {inline(b.text, key)}
              </blockquote>
            );
          case "ul":
            return (
              <ul key={key} className="list-disc space-y-1.5 pl-5">
                {b.items.map((it, j) => (
                  <li key={`${key}-${j}`}>{inline(it, `${key}-${j}`)}</li>
                ))}
              </ul>
            );
          case "img":
            return (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                key={key}
                src={b.src}
                alt={b.alt}
                loading="lazy"
                className="w-full rounded-2xl border border-line"
              />
            );
          case "video":
            return (
              // ⚠️ **A fixed 16:9 box, not a height.** A player that resizes
              // after the page has drawn pushes the paragraph somebody is
              // reading down the screen — on a phone, mid-sentence.
              <div
                key={key}
                className="relative w-full overflow-hidden rounded-2xl border border-line"
                style={{ paddingTop: "56.25%" }}
              >
                <iframe
                  src={`https://www.youtube-nocookie.com/embed/${b.id}`}
                  title="YouTube"
                  className="absolute inset-0 h-full w-full"
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                  allowFullScreen
                  loading="lazy"
                />
              </div>
            );
          default:
            return <p key={key}>{inline(b.text, key)}</p>;
        }
      })}
    </div>
  );
}
