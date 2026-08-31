// Two marks, and deliberately only two.
//
// `*bold*` for the thing that must not be missed, and `` `Text` `` for a button,
// a field or a menu entry the reader has to find on their own screen. That
// second one is the whole reason this exists: instructions that say *press
// Save* and instructions that say press `Saqlash` read the same in a diff and
// completely differently at a counter, because the second one is quoting a
// label the reader is looking at.
//
// ⚠️ **Not a markdown renderer.** A general parser here would let an article
// introduce headings, images and links inside a paragraph, none of which the
// layout is built for — and the day a translator drops a bracket, the page
// renders raw syntax at a customer instead of failing where it can be seen.

const PATTERN = /(\*[^*]+\*|`[^`]+`)/g;

export function Rich({ text }: { text: string }) {
  const parts = text.split(PATTERN).filter(Boolean);
  return (
    <>
      {parts.map((part, i) => {
        if (part.startsWith("*") && part.endsWith("*") && part.length > 2) {
          return (
            <strong key={i} className="font-semibold text-ink">
              {part.slice(1, -1)}
            </strong>
          );
        }
        if (part.startsWith("`") && part.endsWith("`") && part.length > 2) {
          return (
            <span
              key={i}
              className="whitespace-nowrap rounded-md border border-line bg-raised px-1.5 py-0.5 text-[0.92em] font-medium text-ink"
            >
              {part.slice(1, -1)}
            </span>
          );
        }
        return <span key={i}>{part}</span>;
      })}
    </>
  );
}
