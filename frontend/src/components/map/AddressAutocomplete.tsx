"use client";

import { useEffect, useRef, useState } from "react";
import { searchPlaces, type Place } from "@/lib/geocode";

interface Props {
  value: string;
  onTextChange: (text: string) => void;
  onSelect: (place: Place) => void;
  placeholder?: string;
  className?: string;
}

// Address input with debounced Nominatim suggestions. Selecting a suggestion
// reports the picked place (with lat/lng) so the caller can move the map.
export default function AddressAutocomplete({
  value,
  onTextChange,
  onSelect,
  placeholder,
  className,
}: Props) {
  const [results, setResults] = useState<Place[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);
  // Skip searching right after a selection filled the input programmatically.
  const skipNext = useRef(false);
  // Suggestions only appear once the customer actually types.
  const userTyped = useRef(false);

  useEffect(() => {
    if (skipNext.current) {
      skipNext.current = false;
      return;
    }
    const q = value.trim();
    if (q.length < 3) {
      setResults([]);
      setLoading(false);
      return;
    }
    const controller = new AbortController();
    setLoading(true);
    const t = setTimeout(async () => {
      try {
        const places = await searchPlaces(q, controller.signal);
        setResults(places);
        if (userTyped.current) setOpen(true);
      } catch {
        /* aborted or failed */
      } finally {
        setLoading(false);
      }
    }, 400);
    return () => {
      clearTimeout(t);
      controller.abort();
    };
  }, [value]);

  // Close dropdown on outside click.
  useEffect(() => {
    function onDoc(e: MouseEvent) {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

  function pick(p: Place) {
    skipNext.current = true;
    onTextChange(p.text);
    onSelect(p);
    setOpen(false);
    setResults([]);
  }

  return (
    <div ref={boxRef} className="relative">
      <input
        className={className}
        value={value}
        placeholder={placeholder}
        onChange={(e) => {
          userTyped.current = true;
          onTextChange(e.target.value);
        }}
        onFocus={() => results.length > 0 && setOpen(true)}
        autoComplete="off"
      />
      {loading && (
        <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-ink-muted/70">
          qidirilmoqda...
        </span>
      )}
      {open && results.length > 0 && (
        <ul className="absolute z-20 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-line bg-surface shadow-lg">
          {results.map((p, i) => (
            <li key={`${p.lat}-${p.lng}-${i}`}>
              <button
                type="button"
                onClick={() => pick(p)}
                className="block w-full px-3 py-2 text-left text-sm hover:bg-cream"
              >
                {p.text}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
