import { GRACE_DAYS, PLAN_DAYS, PLANS, fa } from "./lib/plans";
import { Footer } from "./ui/Footer";
import { ThemeToggle } from "./ui/ThemeToggle";

export default function HomePage() {
  return (
    <main className="shell">
      <nav className="nav">
        <div className="brand"><span className="brandMark">↑</span> FileTransfer</div>
        <div className="navLinks"><a href="#plans">تعرفه‌ها</a><a href="#help">راهنما</a><a className="login" href="/login">ورود</a><ThemeToggle /></div>
      </nav>

      <section className="hero">
        <div className="eyebrow">ساده، سریع، ایرانی</div>
        <h1>فایل‌هایتان را<br /><span>آسان منتقل کنید.</span></h1>
        <p>آپلود و اشتراک‌گذاری فایل‌های حجیم تا ۵۰ گیگابایت، با لینک مستقیم و امن.</p>

        <div className="uploadCard">
          <div className="uploadIcon">↑</div>
          <h2>فایل خود را اینجا رها کنید</h2>
          <p>یا برای انتخاب فایل کلیک کنید</p>
          <a className="primary" href="/upload">انتخاب فایل</a>
          <small>حداکثر حجم هر فایل: ۵۰ گیگابایت</small>
        </div>

        <div className="trust"><span>✓ بدون نیاز به نصب</span><span>✓ لینک اختصاصی</span><span>✓ سرعت بالا در ایران</span></div>
      </section>

      <section className="features" id="help">
        <article><div className="featureIcon blue">↗</div><h3>اشتراک‌گذاری آسان</h3><p>یک لینک بسازید و فایل را با هرکسی به اشتراک بگذارید.</p></article>
        <article><div className="featureIcon violet">♢</div><h3>امن و قابل کنترل</h3><p>برای هر لینک تاریخ انقضا تعیین کنید و هر زمان خواستید فایل را حذف کنید.</p></article>
        <article><div className="featureIcon green">⌁</div><h3>آپلود قابل ادامه</h3><p>اگر اتصال قطع شد، آپلود را از همان‌جا ادامه دهید.</p></article>
      </section>

      <section className="plans" id="plans">
        <div className="eyebrow">تعرفه‌ها</div>
        <h2>سه پلن ثابت، پیش‌پرداخت</h2>
        <p className="muted">هر پلن {fa(PLAN_DAYS)} روز اعتبار دارد و تمدید خودکار ندارد. قیمت‌ها بدون مالیات بر ارزش افزوده است.</p>
        <div className="planGrid">
          {PLANS.map((p) => (
            <article key={p.id}>
              <h3>{p.name}</h3>
              <div className="price"><strong>{fa(p.priceToman)}</strong> تومان <small>برای {fa(PLAN_DAYS)} روز</small></div>
              <ul>
                <li><b>{fa(p.storageGB)} گیگابایت</b> فضای ذخیره</li>
                <li><b>{fa(p.downloadGB)} گیگابایت</b> ترافیک دانلود</li>
                <li>هر فایل تا <b>{fa(p.maxFileGB)} گیگابایت</b></li>
                <li>نگه‌داری هر فایل تا <b>{fa(p.retentionDays)} روز</b></li>
              </ul>
            </article>
          ))}
        </div>
        <p className="muted">پس از پایان اعتبار، لینک‌ها {fa(GRACE_DAYS)} روز دیگر کار می‌کنند و سپس فایل‌ها حذف می‌شوند. خرید آنلاین پلن به‌زودی فعال می‌شود. جزئیات در <a href="/terms">شرایط استفاده</a> و <a href="/refund">بازگشت وجه</a>.</p>
      </section>

      <Footer />
    </main>
  );
}

