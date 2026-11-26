import { LEGAL_LINKS, SITE } from "../lib/site";

export function Footer() {
  return (
    <footer>
      <nav className="footLinks" aria-label="صفحه‌های قانونی">
        {LEGAL_LINKS.map((l) => <a key={l.href} href={l.href}>{l.label}</a>)}
      </nav>
      <div className="footMeta">
        <span>© ۱۴۰۵ <bdi>{SITE.name}</bdi></span>
        <span>اداره‌کننده: <a href={SITE.operatorUrl} rel="me noopener" target="_blank">{SITE.operator}</a></span>
      </div>
    </footer>
  );
}
