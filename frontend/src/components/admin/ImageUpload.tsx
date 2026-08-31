"use client";

import { useRef, useState } from "react";
import { imageUrl, uploadImage, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";

// Uploads a single image and reports back the stored URL. `value` is the
// current image URL (may be empty).
export default function ImageUpload({
  value,
  onChange,
  hint,
}: {
  value: string;
  onChange: (url: string) => void;
  // ⚠️ **Beside the button, not in a tooltip.** An owner makes these in Canva
  // before opening the panel, and a size they read after uploading is a size
  // they read too late — the picture is already made and the answer is "make
  // it again". Anything the uploader needs decided in advance belongs on the
  // screen where the file is chosen.
  hint?: string;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const t = useAdminT();
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const preview = imageUrl(value);

  async function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setError(null);
    setUploading(true);
    try {
      const url = await uploadImage(file);
      onChange(url);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.upload.failed);
    } finally {
      setUploading(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  return (
    <div className="flex items-start gap-4">
      <div className="h-20 w-20 shrink-0 overflow-hidden rounded-lg border border-line bg-ink/5">
        {preview ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={preview} alt="" className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-xs text-ink-muted/70">
            {t.upload.noImage}
          </div>
        )}
      </div>
      <div className="min-w-0">
        <input
          ref={inputRef}
          type="file"
          accept="image/*"
          onChange={handleFile}
          className="hidden"
        />
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={uploading}
          className="btn-ghost px-3 py-1.5 text-sm disabled:opacity-60"
        >
          {uploading ? t.upload.uploading : preview ? t.upload.change : t.upload.choose}
        </button>
        {value && (
          <button
            type="button"
            onClick={() => onChange("")}
            className="ml-2 text-sm text-ink-muted/70 hover:text-brand"
          >
            {t.common.delete}
          </button>
        )}
        {hint && (
          // ⚠️ `min-w-0` on the column, not a fixed max width: this sits in a
          // two-column grid on a wide screen and one column on a phone, and a
          // flex child will not shrink below its content without it — the
          // paragraph would push the picker out of its half.
          <p className="mt-1.5 text-xs leading-relaxed text-ink-muted">
            {hint}
          </p>
        )}
        {error && <p className="mt-1 text-xs text-brand">{error}</p>}
      </div>
    </div>
  );
}
