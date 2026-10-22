export default function HomePage() {
  return (
    <main className="shell">
      <nav className="nav">
        <div className="brand"><span className="brandMark">↑</span> FileTransfer</div>
        <div className="navLinks"><a href="#plans">تعرفه‌ها</a><a href="#help">راهنما</a><button className="login">ورود</button></div>
      </nav>

      <section className="hero">
        <div className="eyebrow">ساده، سریع، ایرانی</div>
        <h1>فایل‌هایتان را<br /><span>آسان منتقل کنید.</span></h1>
        <p>آپلود و اشتراک‌گذاری فایل‌های حجیم تا ۵۰ گیگابایت، با لینک مستقیم و امن.</p>

        <div className="uploadCard">
          <div className="uploadIcon">↑</div>
          <h2>فایل خود را اینجا رها کنید</h2>
          <p>یا برای انتخاب فایل کلیک کنید</p>
          <button className="primary">انتخاب فایل</button>
          <small>حداکثر حجم هر فایل: ۵۰ گیگابایت</small>
        </div>

        <div className="trust"><span>✓ بدون نیاز به نصب</span><span>✓ لینک اختصاصی</span><span>✓ سرعت بالا در ایران</span></div>
      </section>

      <section className="features" id="help">
        <article><div className="featureIcon blue">↗</div><h3>اشتراک‌گذاری آسان</h3><p>یک لینک بسازید و فایل را با هرکسی به اشتراک بگذارید.</p></article>
        <article><div className="featureIcon violet">♢</div><h3>امن و قابل کنترل</h3><p>برای لینک رمز بگذارید و تاریخ انقضا تعیین کنید.</p></article>
        <article><div className="featureIcon green">⌁</div><h3>آپلود قابل ادامه</h3><p>اگر اتصال قطع شد، آپلود را از همان‌جا ادامه دهید.</p></article>
      </section>

      <footer><span>© ۱۴۰۵ FileTransfer.ir</span><span>ساخته‌شده برای ایران</span></footer>
    </main>
  );
}

