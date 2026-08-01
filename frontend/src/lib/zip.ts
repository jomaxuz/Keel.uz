// A minimal ZIP writer, store-only (no compression).
//
// Downloading forty table cards as forty separate files makes browsers ask
// "allow multiple downloads?" and, if the operator says no, silently drops the
// rest. One archive avoids the question entirely.
//
// Store-only costs nothing here: PNG data is already deflated, so compressing
// it again would save a rounding error while pulling in a whole deflate
// implementation. This is ~100 lines with no dependency.

const CRC_TABLE = (() => {
  const table = new Uint32Array(256);
  for (let i = 0; i < 256; i++) {
    let c = i;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    table[i] = c >>> 0;
  }
  return table;
})();

function crc32(data: Uint8Array): number {
  let c = 0xffffffff;
  for (let i = 0; i < data.length; i++) {
    c = CRC_TABLE[(c ^ data[i]) & 0xff] ^ (c >>> 8);
  }
  return (c ^ 0xffffffff) >>> 0;
}

/** MS-DOS date/time, as ZIP has stored timestamps since 1980. */
function dosStamp(d = new Date()): { time: number; date: number } {
  return {
    time: (d.getHours() << 11) | (d.getMinutes() << 5) | (d.getSeconds() >> 1),
    date:
      ((d.getFullYear() - 1980) << 9) | ((d.getMonth() + 1) << 5) | d.getDate(),
  };
}

interface Entry {
  name: Uint8Array;
  size: number;
  crc: number;
  offset: number;
}

/** Bundles files into a ZIP blob, in the order given. */
export function zipFiles(files: { name: string; data: Uint8Array }[]): Blob {
  const encoder = new TextEncoder();
  const stamp = dosStamp();
  const entries: Entry[] = [];
  const parts: Uint8Array[] = [];
  let offset = 0;

  for (const file of files) {
    const name = encoder.encode(file.name);
    const crc = crc32(file.data);
    const header = new Uint8Array(30 + name.length);
    const view = new DataView(header.buffer);
    view.setUint32(0, 0x04034b50, true); // local file header
    view.setUint16(4, 20, true); // version needed
    view.setUint16(6, 0, true); // flags
    view.setUint16(8, 0, true); // method: stored
    view.setUint16(10, stamp.time, true);
    view.setUint16(12, stamp.date, true);
    view.setUint32(14, crc, true);
    view.setUint32(18, file.data.length, true); // compressed size
    view.setUint32(22, file.data.length, true); // uncompressed size
    view.setUint16(26, name.length, true);
    view.setUint16(28, 0, true); // extra length
    header.set(name, 30);

    parts.push(header, file.data);
    entries.push({ name, size: file.data.length, crc, offset });
    offset += header.length + file.data.length;
  }

  const central: Uint8Array[] = [];
  let centralSize = 0;
  for (const entry of entries) {
    const row = new Uint8Array(46 + entry.name.length);
    const view = new DataView(row.buffer);
    view.setUint32(0, 0x02014b50, true); // central directory header
    view.setUint16(4, 20, true); // version made by
    view.setUint16(6, 20, true); // version needed
    view.setUint16(8, 0, true); // flags
    view.setUint16(10, 0, true); // stored
    view.setUint16(12, stamp.time, true);
    view.setUint16(14, stamp.date, true);
    view.setUint32(16, entry.crc, true);
    view.setUint32(20, entry.size, true);
    view.setUint32(24, entry.size, true);
    view.setUint16(28, entry.name.length, true);
    view.setUint16(30, 0, true); // extra
    view.setUint16(32, 0, true); // comment
    view.setUint16(34, 0, true); // disk number
    view.setUint16(36, 0, true); // internal attrs
    view.setUint32(38, 0, true); // external attrs
    view.setUint32(42, entry.offset, true);
    row.set(entry.name, 46);
    central.push(row);
    centralSize += row.length;
  }

  const end = new Uint8Array(22);
  const endView = new DataView(end.buffer);
  endView.setUint32(0, 0x06054b50, true); // end of central directory
  endView.setUint16(8, entries.length, true);
  endView.setUint16(10, entries.length, true);
  endView.setUint32(12, centralSize, true);
  endView.setUint32(16, offset, true);

  const total =
    parts.reduce((n, p) => n + p.length, 0) +
    central.reduce((n, p) => n + p.length, 0) +
    end.length;
  const out = new Uint8Array(new ArrayBuffer(total));
  let at = 0;
  for (const piece of [...parts, ...central, end]) {
    out.set(piece, at);
    at += piece.length;
  }
  return new Blob([out.buffer], { type: "application/zip" });
}

/** The bytes behind a `data:` URL (what canvas.toDataURL hands back). */
export function dataUrlToBytes(dataUrl: string): Uint8Array {
  const base64 = dataUrl.slice(dataUrl.indexOf(",") + 1);
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

/**
 * The same bytes as a Blob. Goes through a fresh ArrayBuffer because a
 * Uint8Array over a shared buffer is not a `BlobPart` under TypeScript's newer
 * typed-array generics.
 */
export function dataUrlToBlob(dataUrl: string, type = "image/png"): Blob {
  const bytes = dataUrlToBytes(dataUrl);
  const buffer = new ArrayBuffer(bytes.length);
  new Uint8Array(buffer).set(bytes);
  return new Blob([buffer], { type });
}

/**
 * Saves a blob under `name`. The anchor goes into the document before it is
 * clicked and comes out afterwards: a detached anchor does not reliably start a
 * download outside Chrome.
 */
export function saveBlob(blob: Blob, name: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  // Revoking immediately can cancel the download in Safari.
  setTimeout(() => URL.revokeObjectURL(url), 10000);
}
