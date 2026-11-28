import type { Metadata } from "next";
import "@fontsource-variable/vazirmatn";
import "./globals.css";
import { THEME_INIT } from "./ui/ThemeToggle";

export const metadata: Metadata = {
  title: "FileTransfer.ir | انتقال آسان فایل",
  description: "آپلود و اشتراک‌گذاری سریع فایل‌های حجیم",
  // eNAMAD checks this tag on the home page to confirm we own the domain.
  other: { enamad: "15836154" },
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    // The inline script sets data-theme before hydration, so React must not warn about it.
    <html lang="fa" dir="rtl" suppressHydrationWarning>
      <head><script dangerouslySetInnerHTML={{ __html: THEME_INIT }} /></head>
      <body>{children}</body>
    </html>
  );
}

