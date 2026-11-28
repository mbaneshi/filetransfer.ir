"use client";

import { useSession } from "../lib/session";

// The home page's upload card: a real way in when logged in, a login prompt when not.
export function UploadCta() {
  const session = useSession();
  const out = session === "out";
  return (
    <div className="uploadCard">
      <div className="uploadIcon">↑</div>
      <h2>{out ? "برای آپلود وارد شوید" : "فایل خود را آپلود کنید"}</h2>
      <p>{out ? "با شماره موبایل وارد شوید؛ کد تأیید برایتان پیامک می‌شود." : "فایل را انتخاب کنید و لینک اشتراک بسازید."}</p>
      <a className="primary" href={out ? "/login?next=/upload" : "/upload"}>{out ? "ورود و آپلود" : "انتخاب فایل"}</a>
      <small>حداکثر حجم هر فایل: ۵۰ گیگابایت</small>
    </div>
  );
}
