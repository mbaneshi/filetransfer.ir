import type { Metadata } from "next";
import "@fontsource-variable/vazirmatn";
import "./globals.css";

export const metadata: Metadata = {
  title: "FileTransfer.ir | انتقال آسان فایل",
  description: "آپلود و اشتراک‌گذاری سریع فایل‌های حجیم",
  // eNAMAD checks this tag on the home page to confirm we own the domain.
  other: { enamad: "15836154" },
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="fa" dir="rtl">
      <body>{children}</body>
    </html>
  );
}

