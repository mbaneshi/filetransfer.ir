import type { ReactNode } from "react";
import { SITE } from "../lib/site";
import { Footer } from "./Footer";
import { BackLink, SiteNav } from "./SiteNav";

export function LegalPage({ title, children }: { title: string; children: ReactNode }) {
  return (
    <main className="shell">
      <SiteNav />
      <article className="prose">
        <BackLink />
        <h1>{title}</h1>
        <p className="muted">نسخهٔ {SITE.termsVersion}، به‌روزرسانی: {SITE.termsDate}</p>
        {children}
      </article>
      <Footer />
    </main>
  );
}
