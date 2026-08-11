// Tokens shared by every band the constructor can draw.
//
// One map rather than one per file: `tone` is a value the console offers on most
// bands and the schema validates in one place, so two renderers reading it from
// two lists is a way for "charcoal" to mean a dark band in one band and nothing
// in another. Written out rather than composed, because Tailwind can only see
// class names that appear literally in the source.

export const TONE: Record<string, string> = {
  "": "",
  surface: "bg-surface",
  raised: "bg-raised",
  charcoal: "bg-charcoal text-white",
  brand: "bg-brand text-white",
};
