"use client";

import { logout, useSession } from "../lib/session";
import { ThemeToggle } from "./ThemeToggle";

// The one header every page shares: home, plans, help, then the account links for the current session.
export function SiteNav() {
  const session = useSession();
  return (
    <nav className="nav">
      <a href="/" className="brand"><span className="brandMark">↑</span> FileTransfer</a>
      <div className="navLinks">
        <a href="/#plans">تعرفه‌ها</a>
        <a href="/#help">راهنما</a>
        {session === "in" ? (
          <>
            <a className="login" href="/upload">فایل‌های من</a>
            <button className="login" onClick={logout}>خروج</button>
          </>
        ) : (
          <a className="login" href="/login" aria-hidden={session === "loading"} style={session === "loading" ? { visibility: "hidden" } : undefined}>ورود</a>
        )}
        <ThemeToggle />
      </div>
    </nav>
  );
}

// Goes back where the visitor came from on this site, or home when they landed here directly.
export function BackLink() {
  function back(e: React.MouseEvent) {
    if (document.referrer.startsWith(window.location.origin) && window.history.length > 1) {
      e.preventDefault();
      window.history.back();
    }
  }
  return <a className="back" href="/" onClick={back}>→ بازگشت</a>;
}
