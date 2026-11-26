import type { ReactNode } from "react";
import { SITE } from "../lib/site";
import { Footer } from "./Footer";

export function LegalPage({ title, children }: { title: string; children: ReactNode }) {
  return (
    <main className="shell">
      <nav className="nav">
        <a href="/" className="brand"><span className="brandMark">↑</span> FileTransfer</a>
        <div className="navLinks"><a href="/#plans">تعرفه‌ها</a><a className="login" href="/login">ورود</a></div>
      </nav>
      <article className="prose">
        <h1>{title}</h1>
        <p className="muted">نسخهٔ {SITE.termsVersion}، به‌روزرسانی: {SITE.termsDate}</p>
        {children}
      </article>
      <Footer />
    </main>
  );
}
