import { api } from "./api";

type Init = { id: string; partSize: number; partCount: number };
type Status = { status: string; partSize: number; partCount: number; uploaded?: { number: number; size: number }[] };

export type UploadHandle = { done: Promise<{ id: string }>; cancel: () => void };

const CONCURRENCY = 3;
const URLS_PER_CALL = 100;
const RETRIES = 3;

// Remembers the in-flight upload per file so a reload can resume it.
const keyOf = (f: File) => `ft-upload:${f.name}:${f.size}:${f.lastModified}`;
const remember = (f: File, id?: string) => {
  try {
    id ? localStorage.setItem(keyOf(f), id) : localStorage.removeItem(keyOf(f));
  } catch {}
};
const recall = (f: File) => {
  try {
    return localStorage.getItem(keyOf(f));
  } catch {
    return null;
  }
};

function putPart(url: string, blob: Blob, signal: AbortSignal, onBytes: (loaded: number) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", url);
    xhr.upload.onprogress = (e) => onBytes(e.loaded);
    xhr.onload = () => (xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error(`part ${xhr.status}`)));
    xhr.onerror = () => reject(new Error("network"));
    signal.addEventListener("abort", () => (xhr.abort(), reject(new DOMException("aborted", "AbortError"))), { once: true });
    xhr.send(blob);
  });
}

export function startUpload(file: File, onProgress: (sent: number, total: number) => void): UploadHandle {
  const ctl = new AbortController();
  let uploadId: string | null = null;

  const run = async () => {
    let init: Init | Status | null = null;
    let id = recall(file);
    if (id) {
      try {
        const s = await api<Status>(`/uploads/${id}`);
        if (s.status === "uploading") init = s;
      } catch {}
      if (!init) id = null;
    }
    if (!init || !id) {
      const c = await api<Init>("/uploads", {
        method: "POST",
        json: { name: file.name, size: file.size, contentType: file.type },
      });
      id = c.id;
      init = c;
    }
    uploadId = id;
    remember(file, id);

    const { partSize, partCount } = init;
    const done = new Set<number>();
    const inFlight = new Map<number, number>();
    let sent = 0;
    for (const p of (init as Status).uploaded ?? []) {
      done.add(p.number);
      sent += p.size;
    }
    const report = () => onProgress(Math.min(file.size, sent + [...inFlight.values()].reduce((a, b) => a + b, 0)), file.size);
    report();

    const pending = Array.from({ length: partCount }, (_, i) => i + 1).filter((n) => !done.has(n));
    const urls = new Map<number, string>();
    const fetchUrls = async (nums: number[]) => {
      const r = await api<{ urls: Record<string, string> }>(`/uploads/${id}/parts`, { method: "POST", json: { parts: nums } });
      for (const [k, v] of Object.entries(r.urls)) urls.set(Number(k), v);
    };

    const worker = async () => {
      for (;;) {
        const n = pending.shift();
        if (n === undefined) return;
        if (!urls.has(n)) await fetchUrls([n, ...pending.slice(0, URLS_PER_CALL - 1).filter((m) => !urls.has(m))]);
        const blob = file.slice((n - 1) * partSize, Math.min(n * partSize, file.size));
        for (let attempt = 1; ; attempt++) {
          try {
            await putPart(urls.get(n)!, blob, ctl.signal, (l) => (inFlight.set(n, l), report()));
            break;
          } catch (e) {
            inFlight.delete(n);
            if ((e as Error).name === "AbortError" || attempt >= RETRIES) throw e;
            await new Promise((r) => setTimeout(r, 1000 * attempt));
          }
        }
        inFlight.delete(n);
        sent += blob.size;
        report();
      }
    };

    await Promise.all(Array.from({ length: Math.min(CONCURRENCY, pending.length) }, worker));
    await api(`/uploads/${id}/complete`, { method: "POST" });
    remember(file);
    return { id };
  };

  return {
    done: run(),
    cancel: () => {
      ctl.abort();
      if (uploadId) {
        api(`/uploads/${uploadId}`, { method: "DELETE" }).catch(() => {});
        remember(file);
      }
    },
  };
}
