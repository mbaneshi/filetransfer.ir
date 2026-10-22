import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "FileTransfer.ir | انتقال آسان فایل",
  description: "آپلود و اشتراک‌گذاری سریع فایل‌های حجیم",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="fa" dir="rtl">
      <body>{children}</body>
    </html>
  );
}

