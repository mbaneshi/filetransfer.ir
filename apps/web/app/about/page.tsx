import type { Metadata } from "next";
import { SITE } from "../lib/site";
import { LegalPage } from "../ui/LegalPage";

export const metadata: Metadata = { title: "درباره و تماس | FileTransfer.ir" };

export default function AboutPage() {
  return (
    <LegalPage title="درباره و تماس">
      <h2>این سرویس چیست</h2>
      <p><bdi>FileTransfer.ir</bdi> سرویسی ایرانی برای فرستادن فایل‌های حجیم است. فایل را بارگذاری می‌کنید، لینک می‌گیرید و آن را برای هر کس که بخواهید می‌فرستید. فایل‌ها روی زیرساخت ابری داخل ایران نگه‌داری می‌شوند.</p>

      <h2>چه کسی آن را اداره می‌کند</h2>
      <p>این سرویس را <a href={SITE.operatorUrl} rel="me noopener" target="_blank">{SITE.operator}</a> ساخته و اداره می‌کند. سایت شخصی: <a href={SITE.operatorUrl} rel="me noopener" target="_blank"><bdi>mbaneshi.ir</bdi></a></p>

      <h2>تماس</h2>
      <ul>
        {SITE.address && <li>نشانی: {SITE.address}{SITE.postalCode && <>، کد پستی <bdi>{SITE.postalCode}</bdi></>}</li>}
        {SITE.phone && <li>تلفن: <a href={`tel:${SITE.phone}`}><bdi>{SITE.phone}</bdi></a></li>}
        {SITE.email && <li>ایمیل: <a href={`mailto:${SITE.email}`}><bdi>{SITE.email}</bdi></a></li>}
        <li>سایت اداره‌کننده: <a href={SITE.operatorUrl} rel="me noopener" target="_blank"><bdi>mbaneshi.ir</bdi></a></li>
      </ul>

      <h2>صفحه‌های مرتبط</h2>
      <p><a href="/terms">شرایط استفاده</a>، <a href="/privacy">حریم خصوصی</a>، <a href="/refund">بازگشت وجه</a>، <a href="/report">گزارش تخلف و شکایت</a></p>
    </LegalPage>
  );
}
