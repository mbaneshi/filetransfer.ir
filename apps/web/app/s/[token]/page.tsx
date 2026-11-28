"use client";

import { use, useEffect, useState } from "react";
import { api, errorMessage, formatBytes } from "../../lib/api";
import { Footer } from "../../ui/Footer";
import { BackLink, SiteNav } from "../../ui/SiteNav";

export default function SharePage({ params }: { params: Promise<{ token: string }> }) {
  const { token } = use(params);
  const [info, setInfo] = useState<{ name: string; sizeBytes: number } | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ name: string; sizeBytes: number }>(`/shares/${encodeURIComponent(token)}`).then(setInfo, (e) => setError(errorMessage(e)));
  }, [token]);

  return (
    <main className="shell">
      <SiteNav />
      <div className="narrow">
      <BackLink />
      <div className="card">
        {error ? <div className="error" role="alert">{error}</div> : !info ? <p>در حال بارگذاری…</p> : (
          <>
            <div className="uploadIcon">↓</div>
            <h2 className="name">{info.name}</h2>
            <p>{formatBytes(info.sizeBytes)}</p>
            <a className="primary wide" href={`/api/v1/shares/${encodeURIComponent(token)}/download`}>دانلود فایل</a>
          </>
        )}
      </div>
      </div>
      <Footer />
    </main>
  );
}
