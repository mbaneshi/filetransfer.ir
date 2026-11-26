"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, api, daysUntil, errorMessage, formatBytes, formatDate } from "../lib/api";
import { PLANS, fa } from "../lib/plans";
import { startUpload, type UploadHandle } from "../lib/upload";
import { ThemeToggle } from "../ui/ThemeToggle";

type FileRow = { id: string; name: string; sizeBytes: number; createdAt: string; expiresAt?: string };
type Account = {
  plansEnforced: boolean;
  storageUsedBytes: number;
  trafficUsedBytes?: number;
  plan: null | {
    id: string; active: boolean; expiresAt: string; filesDeletedAt: string;
    storageBytes: number; downloadBytes: number; maxFileBytes: number; retentionDays: number;
  };
};

const HOUR = 3_600_000;
const SHARE_DAYS = [1, 3, 7, 14, 30]; // the API accepts 1 hour to 30 days
const DEFAULT_SHARE_DAYS = 7;

// Link lifetimes on offer for a file: none may outlive the file, and "until deletion" covers the rest.
function shareOptions(f: FileRow) {
  if (!f.expiresAt) return SHARE_DAYS.map((d) => ({ hours: d * 24, label: `${fa(d)} روز` }));
  const left = Math.max(1, Math.floor((new Date(f.expiresAt).getTime() - Date.now()) / HOUR));
  const opts = SHARE_DAYS.filter((d) => d * 24 < left).map((d) => ({ hours: d * 24, label: `${fa(d)} روز` }));
  return [...opts, { hours: Math.min(left, 30 * 24), label: "تا حذف فایل" }];
}
const defaultHours = (f: FileRow) => {
  const opts = shareOptions(f);
  return (opts.find((o) => o.hours === DEFAULT_SHARE_DAYS * 24) ?? opts[opts.length - 1]).hours;
};

function Meter({ label, used, total }: { label: string; used: number; total: number }) {
  const pct = Math.min(100, Math.round((used / total) * 100));
  return (
    <div className="meter">
      <div><span>{label}</span><span>{formatBytes(used)} از {formatBytes(total)}</span></div>
      <div className={pct >= 100 ? "bar full" : "bar"} role="progressbar" aria-label={label} aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}><i style={{ width: `${pct}%` }} /></div>
    </div>
  );
}

function AccountPanel({ account }: { account: Account }) {
  const p = account.plan;
  if (!p) {
    return (
      <section className="account">
        <header><h3>طرح فعالی ندارید</h3><a className="login" href="/#plans">دیدن طرح‌ها</a></header>
        <p className="muted">برای آپلود به یک طرح نیاز دارید. تا راه‌اندازی پرداخت آنلاین، طرح‌ها را پشتیبانی فعال می‌کند. راه تماس در صفحه‌ی <a href="/about">درباره‌ی ما</a> است.</p>
      </section>
    );
  }
  const name = PLANS.find((x) => x.id === p.id)?.name ?? p.id;
  return (
    <section className="account">
      <header>
        <h3>طرح <b>{name}</b></h3>
        <span className="muted">{p.active ? `${fa(daysUntil(p.expiresAt))} روز مانده، تا ${formatDate(p.expiresAt)}` : "اشتراک به پایان رسیده"}</span>
      </header>
      {!p.active && <p className="warn">آپلود متوقف شده است. لینک‌ها تا {formatDate(p.filesDeletedAt)} کار می‌کنند و پس از آن همه‌ی فایل‌ها حذف می‌شوند. برای ادامه، طرح را تمدید کنید.</p>}
      <Meter label="فضای ذخیره‌سازی" used={account.storageUsedBytes} total={p.storageBytes} />
      <Meter label="ترافیک دانلود این دوره" used={account.trafficUsedBytes ?? 0} total={p.downloadBytes} />
      <p className="muted">هر فایل تا {formatBytes(p.maxFileBytes)}؛ نگهداری هر فایل {fa(p.retentionDays)} روز.</p>
    </section>
  );
}

export default function UploadPage() {
  const [files, setFiles] = useState<FileRow[] | null>(null);
  const [progress, setProgress] = useState<{ name: string; sent: number; total: number } | null>(null);
  const [error, setError] = useState("");
  const [links, setLinks] = useState<Record<string, string>>({});
  const [account, setAccount] = useState<Account | null>(null);
  const [hours, setHours] = useState<Record<string, number>>({});
  const handle = useRef<UploadHandle | null>(null);
  const input = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    try {
      setFiles((await api<{ files: FileRow[] }>("/files")).files);
      // The panel is extra: an API without plans has no /account, and the page still works.
      setAccount(await api<Account>("/account").catch(() => null));
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

  async function share(f: FileRow) {
    try {
      const r = await api<{ url: string }>(`/files/${f.id}/shares`, { method: "POST", json: { expiresInHours: hours[f.id] ?? defaultHours(f) } });
      setLinks((l) => ({ ...l, [f.id]: r.url }));
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
        <div className="navLinks"><button className="login" onClick={logout}>خروج</button><ThemeToggle /></div>
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

      {account?.plansEnforced && <AccountPanel account={account} />}

      <section className="list">
        <h3>فایل‌های من</h3>
        {files === null ? <p className="muted">در حال بارگذاری…</p> : files.length === 0 ? <p className="muted">هنوز فایلی آپلود نکرده‌اید.</p> : files.map((f) => (
          <div className="row" key={f.id}>
            <div className="grow"><b className="name">{f.name}</b><small>{formatBytes(f.sizeBytes)}{f.expiresAt && ` · حذف در ${formatDate(f.expiresAt)}`}</small></div>
            {links[f.id] ? <input className="field ltr small" readOnly value={links[f.id]} onFocus={(e) => e.target.select()} /> : (
              <>
                <select className="field small pick" aria-label="مدت اعتبار لینک" value={hours[f.id] ?? defaultHours(f)} onChange={(e) => setHours((h) => ({ ...h, [f.id]: Number(e.target.value) }))}>
                  {shareOptions(f).map((o) => <option key={o.hours} value={o.hours}>{o.label}</option>)}
                </select>
                <button className="login" onClick={() => share(f)}>ساخت لینک</button>
              </>
            )}
            <button className="login danger" onClick={() => remove(f.id)}>حذف</button>
          </div>
        ))}
      </section>
    </main>
  );
}
