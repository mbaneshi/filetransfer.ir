"use client";

import { use, useEffect, useState } from "react";
import { ApiError, api, errorMessage, formatBytes } from "../../lib/api";

// blocked is set when the owner's plan stops the download (plan_expired or traffic_exhausted).
type Info = { name: string; sizeBytes: number; blocked?: string };

export default function SharePage({ params }: { params: Promise<{ token: string }> }) {
  const { token } = use(params);
  const [info, setInfo] = useState<Info | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api<Info>(`/shares/${encodeURIComponent(token)}`).then(setInfo, (e) => setError(errorMessage(e)));
  }, [token]);

  return (
    <main className="shell narrow">
      <a href="/" className="brand"><span className="brandMark">↑</span> FileTransfer</a>
      <div className="card">
        {error ? <div className="error" role="alert">{error}</div> : !info ? <p>در حال بارگذاری…</p> : (
          <>
            <div className="uploadIcon">↓</div>
            <h2 className="name">{info.name}</h2>
            <p>{formatBytes(info.sizeBytes)}</p>
            {info.blocked ? (
              <div className="error" role="alert">{errorMessage(new ApiError(403, info.blocked))}</div>
            ) : (
              <a className="primary wide" href={`/api/v1/shares/${encodeURIComponent(token)}/download`}>دانلود فایل</a>
            )}
          </>
        )}
      </div>
    </main>
  );
}
