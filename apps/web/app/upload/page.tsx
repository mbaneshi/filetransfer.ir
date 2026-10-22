"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, api, errorMessage, formatBytes } from "../lib/api";
import { startUpload, type UploadHandle } from "../lib/upload";

type FileRow = { id: string; name: string; sizeBytes: number; createdAt: string };

export default function UploadPage() {
  const [files, setFiles] = useState<FileRow[] | null>(null);
  const [progress, setProgress] = useState<{ name: string; sent: number; total: number } | null>(null);
  const [error, setError] = useState("");
  const [links, setLinks] = useState<Record<string, string>>({});
  const handle = useRef<UploadHandle | null>(null);
  const input = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    try {
      setFiles((await api<{ files: FileRow[] }>("/files")).files);
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) window.location.href = "/login";
      else setError(errorMessage(e));
    }
  }, []);
  useEffect(() => { load(); }, [load]);

  async function pick(file: File | undefined) {
    if (!file || progress) return;
    setError("");
    setProgress({ name: file.name, sent: 0, total: file.size });
    const h = startUpload(file, (sent, total) => setProgress({ name: file.name, sent, total }));
    handle.current = h;
    try {
      await h.done;
      await load();
    } catch (e) {
      if ((e as Error).name !== "AbortError") setError(errorMessage(e));
    } finally {
      handle.current = null;
      setProgress(null);
      if (input.current) input.current.value = "";
    }
  }

  async function share(id: string) {
    try {
      const r = await api<{ url: string }>(`/files/${id}/shares`, { method: "POST" });
      setLinks((l) => ({ ...l, [id]: r.url }));
      navigator.clipboard?.writeText(r.url).catch(() => {});
    } catch (e) {
      setError(errorMessage(e));
    }
  }

  async function remove(id: string) {
    try {
      await api(`/files/${id}`, { method: "DELETE" });
      await load();
    } catch (e) {
      setError(errorMessage(e));
    }
  }

  async function logout() {
    await api("/auth/logout", { method: "POST" }).catch(() => {});
    window.location.href = "/";
  }

  const pct = progress ? Math.floor((progress.sent / progress.total) * 100) : 0;

  return (
    <main className="shell">
      <nav className="nav">
        <a href="/" className="brand"><span className="brandMark">↑</span> FileTransfer</a>
        <button className="login" onClick={logout}>خروج</button>
      </nav>

      <section
        className="uploadCard dash"
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => { e.preventDefault(); pick(e.dataTransfer.files[0]); }}
      >
        {progress ? (
          <>
            <h2>{progress.name}</h2>
            <div className="bar" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}><i style={{ width: `${pct}%` }} /></div>
            <p>{new Intl.NumberFormat("fa-IR").format(pct)}٪ — {formatBytes(progress.sent)} از {formatBytes(progress.total)}</p>
            <button className="login" onClick={() => handle.current?.cancel()}>لغو</button>
          </>
        ) : (
          <>
            <div className="uploadIcon">↑</div>
            <h2>فایل خود را اینجا رها کنید</h2>
            <p>یا برای انتخاب فایل کلیک کنید. اگر آپلود قطع شد، همان فایل را دوباره انتخاب کنید تا ادامه پیدا کند.</p>
            <button className="primary" onClick={() => input.current?.click()}>انتخاب فایل</button>
          </>
        )}
        <input ref={input} type="file" hidden onChange={(e) => pick(e.target.files?.[0])} />
      </section>

      {error && <div className="error center" role="alert">{error}</div>}

      <section className="list">
        <h3>فایل‌های من</h3>
        {files === null ? <p className="muted">در حال بارگذاری…</p> : files.length === 0 ? <p className="muted">هنوز فایلی آپلود نکرده‌اید.</p> : files.map((f) => (
          <div className="row" key={f.id}>
            <div className="grow"><b className="name">{f.name}</b><small>{formatBytes(f.sizeBytes)}</small></div>
            {links[f.id] ? <input className="field ltr small" readOnly value={links[f.id]} onFocus={(e) => e.target.select()} /> : <button className="login" onClick={() => share(f.id)}>ساخت لینک</button>}
            <button className="login danger" onClick={() => remove(f.id)}>حذف</button>
          </div>
        ))}
      </section>
    </main>
  );
}
