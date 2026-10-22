# ArvanCloud answers for filetransfer.ir (public file host, up to 50 GB/file)

> Fetched **2026-10-22** by the `arv` session, read-only, using a real browser (the plain
> HTTP fetch tool only sees Arvan's "transferring to the site…" interstitial, so
> `www.arvancloud.ir` pages must be rendered with a JS-capable browser).
> Persian page text is authoritative; amounts are in **toman** unless noted.
> Requested by the `file` session (project `~/filetransfer.ir`). Every number below
> carries its page URL; all pages were fetched on 2026-10-22 unless a different date is given.

Legend: **CONFIRMED** (a fetched page states it) / **PARTIAL** / **NOT FOUND** (with the exact terms searched, per repo convention #3 in `CLAUDE.md`).

---

## Q1 — Storage rate: 3,000 vs 1,400 toman/GB — **CONFIRMED, and the page contradicts itself**

Source: <https://www.arvancloud.ir/fa/pricing/cloud-storage> (fetched 2026-10-22).

Package table (monthly):

| Package | Price | Included storage | Included traffic | Included read/write requests |
|---|---|---|---|---|
| Base (پایه) | free | 5 GB | 20 GB | 10,000 |
| Growth (رشد) | 1,500,000 | 500 GB | 2 TB | 20,000 |
| Professional (حرفه‌ای) | 15,000,000 | 5 TB | 20 TB | 50,000 |
| Enterprise (سازمانی) | custom | unlimited | unlimited | 100,000 |

Overage ("مصرف مازاد") section of the **same page** shows a *tiered* standard-storage rate:

- up to 5 GB: free
- 5 GB – 5 TB: **3,000** toman/GB
- 5 – 50 TB: 2,600
- 50 – 500 TB: 2,300
- over 500 TB: 2,000
- "Hiops" storage class: 9,000 toman/GB

But the FAQ answer "منظور از مصرف مازاد و پرداخت PAYG در فضای ابری چیست؟" on that same page says
overage is charged "**۱,۴۰۰ تومان** به‌ازای هر گیگابایت فضای ذخیره‌سازی و **۸۵۰ تومان** به‌ازای هر گیگابایت ترافیک".

So the 3,000 figure is the tiered PAYG table and 1,400 is the FAQ's flat overage number. The page
does not say which applies to a package's overage vs. a plain PAYG account. The download-traffic
850 matches in both places. **Do not model with either number as settled**: ask Arvan sales/support
(or buy Growth on a small test and read the wallet deduction). The two tiers differ by 2.1x.

Other facts from the same page:
- Buying Growth or Professional requires wallet credit of ≥ 14 days of the package, i.e. **50 % of the package price** (FAQ "برای خرید بسته‌های فضای ابری باید چه‌قدر در کیف پولم اعتبار داشته باشم؟").
- Package changes are limited to **2 per month** (FAQ "آیا همیشه امکان خرید بسته‌های بالاتر وجود دارد؟").
- The free Base package has **no time limit** (FAQ "آیا بسته‌ی رایگان فضای ابری محدودیت زمانی دارد؟"). The terms say free-tier services get no SLA and may be changed or removed (Terms, section "استفاده از خدمات رایگان").

## Q2 — CDN vs direct S3 egress billing — **PARTIAL**

- Object storage download traffic, from the storage pricing page above: up to 20 GB free; 20 GB – 20 TB **850**/GB; 20 – 200 TB 700; 200 TB – 2 PB 600; over 2 PB 500.
- The storage docs say buckets are integrated with Arvan's CDN **by default** and that CDN caching only applies when you use **virtual-host-style URLs** (`<bucket>.s3.<region>.arvanstorage.ir`): <https://docs.arvancloud.ir/fa/object-storage/limits/> (local mirror `docs/upstream/pages/object-storage__limits.txt`, fetched 2026-08-26).
- Standalone CDN billing: <https://www.arvancloud.ir/fa/pricing/cdn> (2026-10-22). Traffic is listed as "unlimited" in every package, but "according to the fair-usage policy; overage is PAYG".
  - Overage: "ترافیک دریافتی ایران و خارج" beyond fair usage: **1,500** toman/GB. "ترافیک ارسالی" (sent): free.
  - Fair usage (FAQ on that page): Base **8 GB/day**, Growth **50 GB/day**, Professional **500 GB/day** of "ترافیک دریافتی (از سرور لبه)"; sent traffic from the edge is unlimited in all packages. HTTP(S) request fair-usage cap: **1 billion/month** in every package.
  - CDN request overage per 10,000: Iran 1–5 billion = 96, over 5 billion = 48; abroad 1–5 billion = 144, over = 96.
  - **Ambiguity I could not resolve:** the page does not define whether "دریافتی (از سرور لبه)" means origin→edge or edge→user. The wording "sent from the edge is unlimited" suggests user downloads are the unlimited direction, but this should be confirmed with Arvan before you rely on it for a download-heavy business.
- **NOT FOUND:** whether bucket traffic served *through* the CDN is billed on the storage meter, the CDN meter, or both. Terms searched (mirror + live pages): "ترافیک CDN فضای ابری", "free egress", "ترافیک رایگان CDN فضای ابری". Ask support.

## Q3 — Multipart parts as write requests — **NOT FOUND**

- Pricing page lists write/read request tiers (per 10,000 requests) but never defines "write request". Write overage: Base >10,000 = **6,000**/10k; Growth 20k–1M = 6,000, 1–20M = 5,500, 20–50M = 5,000, 50–200M = 4,000, >200M = 3,000; Professional >50,000 = 6,100; Enterprise 100k–1M = 6,100, then 5,500 / 5,000 / 4,000 / 3,000. Read overage: 500/10k, falling to 400 / 350 / 300 on higher volume tiers. (Source: <https://www.arvancloud.ir/fa/pricing/cloud-storage>, 2026-10-22.)
- The S3 multipart docs (<https://docs.arvancloud.ir/fa/developer-tools/sdk/object-storage/multipart-upload/>) describe initiate / upload-part / complete / abort, but say nothing about billing.
- Terms searched: "درخواست نوشتن", "multipart", "بخش", "UploadPart" across the object-storage pricing/docs pages. Practical assumption to test: each UploadPart is a PUT, so likely counted as a write; for a 50 GB file at 64 MB parts that is ~800 writes ≈ 0.8 of one 10k-write billing unit (≈ 480 toman at 6,000/10k), which is small. Treat as unverified.
- Also: abandoned multipart uploads **still consume storage until aborted** (multipart docs). Lifecycle rules can delete incomplete uploads (<https://docs.arvancloud.ir/fa/object-storage/buckets/lifecycle/>).

## Q6 — CDN for 50 GB files: max cacheable size, Range, signed URLs — **PARTIAL (the cache-size limit is the finding)**

- **Max cacheable file size:** by default **100 MB** (104,857,600 bytes), raisable to **128 MB**; per plan **Growth 256 MB, Professional 512 MB, Enterprise up to 2,048 MB**. Source: <https://docs.arvancloud.ir/fa/cdn/caching/> (mirror `cdn__caching.txt`, fetched 2026-08-26). **So a 50 GB file can never sit in the CDN cache on any plan.** Large downloads would pass through the CDN uncached (or go direct to the bucket).
- **NOT FOUND:** Range/partial-content behaviour, and CDN-edge signed/token URLs. Terms searched in the 429-page mirror: "Range", "byte-range", "secure link", "لینک امن", "لینک موقت", "signed url", "توکن امنیتی". Zero relevant hits.
- Presigned S3 URLs are supported and documented with the standard AWS SDKs (upload and download pages; examples use 1–24 h expiry); max expiry and SigV4/V2 are **not stated**: <https://docs.arvancloud.ir/fa/developer-tools/sdk/object-storage/download-presigned/> and `.../upload-presigned/`.
- Design implication: a presigned-URL, direct-to-bucket download path for big files does not depend on any of the CDN unknowns.

## Q8 — Antivirus — **CONFIRMED price, plan gate and scan modes; blocking behaviour NOT FOUND**

- Price: **10,000 toman per GB** (add-on). Source: <https://www.arvancloud.ir/fa/pricing/cloud-storage> (2026-10-22), "قابلیت‌های بیش‌تر → آنتی‌ویروس".
- Plan gate: add-on on Growth, Professional, Enterprise; **not available on Base**. (Same page feature table; docs say "پلن‌های رشد و بالاتر".)
- Billing: "هزینه‌ی هر آنتی‌ویروس جداگانه و بر اساس حجم داده‌های اسکن‌شده محاسبه می‌شود", i.e. **per engine, per GB scanned**. Engines: eScan, F-Secure, ESET, Kaspersky, Padvish. Source: <https://docs.arvancloud.ir/fa/object-storage/buckets/antivirus/> (mirror fetched 2026-08-26).
- Cost arithmetic (mine, not Arvan's): scanning one 50 GB file with one engine = 50 × 10,000 = **500,000 toman**. Scanning every upload of a public file host will dominate your cost ledger; scan only on-demand/flagged files or small types.
- Modes: manual bucket scan, scan selected files, or "اسکن آنی" (instant scan automatically after upload, must be switched on). Per-object status and a CSV report exist.
- **NOT FOUND:** whether a file is downloadable while its scan is pending, and any size ceiling per scanned file. Terms searched: "قرنطینه", "تا پایان اسکن", "دانلود", "حجم" in the antivirus page. The docs frame scanning as "before publishing", which suggests it does **not** block by itself; you would need to hold the file private until the scan status flips.
- Arvan's AUP also reserves the right to **quarantine or delete** malicious content found in storage (see Q11).

## Q11 — ToS / AUP / takedown / KYC / eNAMAD — **PARTIAL**

Sources (fetched 2026-10-22, both "last updated Khordad 1405 / ~June 2026"):
terms <https://www.arvancloud.ir/fa/legal/terms>, AUP <https://www.arvancloud.ir/fa/legal/acceptable-use-policy>.

- **Governing law:** Arvan (Noyan Abr Arvan Co.) operates under Iranian law. Distributing content criminal under Iranian law is prohibited; on learning of it Arvan blocks it and **does not refund**. Named prohibited categories include immoral/sexual imagery, narcotics, WMD, military activity, terrorist/separatist groups, "any activity contrary to Iranian law" (terms §"قوانین ایران").
- **Takedown / abuse process:** there is **no notice-and-takedown procedure for third-party copyright claims** on these pages. Instead: Arvan may, "on its own judgement or on requests/complaints/reports from legal bodies and competent organisations", hand over user/IP info or stop service; and on "valid complaints" may limit, block or terminate the account or service; users must provide identifying information on request (terms §"حقوق مولف…", AUP §"گزارش نقض سیاست").
- **AUP content rules relevant to a file host:** bans infringement of third-party IP, immoral/pornographic/violent/racist/hateful content, malware/viruses, phishing, and "excessive or abnormal" resource use (AUP). Arvan "may quarantine or delete" infected/malicious content in storage and may suspend without prior notice and report suspicious activity to legal bodies.
- **Liability:** a customer serving end users is responsible for compliance with the laws and for any third-party rights violations (terms §"مسوولیت در قبال کاربران نهایی"). Arvan does no backups for you and disclaims data loss (terms §"تهیه نسخه پشتیبان", §"شرایط و روند قطع سرویس").
- **Fair usage:** monthly per-account; if exceeded, service can be limited or cut "immediately without consent" (terms §"سیاست مصرف منصفانه"). Relevant for a 50 GB-file download business.
- **KYC:** account details must match official ID documents (terms §"حساب کاربری"). Refunds require national-ID image, matching bank card and a one-time refund per account (min 500,000 toman, 2 weeks old, ≥ 7 working days). No separate document-upload KYC step for storage is described.
- **Free tier:** no SLA and no compensation (terms §"استفاده از خدمات رایگان"); credits for outages only for SLA-covered paid users (§"جبران خسارت").
- **eNAMAD: NOT FOUND on Arvan's pages.** Terms searched in terms + AUP + local mirror: "اینماد", "enamad", "نماد اعتماد", "مجوز", "پروانه". Nothing requires it of customers. eNAMAD is an Iranian site-trust badge issued outside Arvan; I could not read `enamad.ir` from this machine (connection refused), so I have **no verified requirements** to give you. Check <https://enamad.ir> directly and ask a lawyer about the content-liability and licensing duties of an Iranian public file host (the Q11 questions beyond Arvan's own terms are outside this knowledge base).

## Redis — **CONFIRMED: not a managed service today; self-run on Cloud Container**

- The managed database product supports **MySQL and PostgreSQL only**; the docs say MongoDB and Redis "will become possible in the future" ("در آینده … امکان‌پذیر خواهد شد"). Source: <https://docs.arvancloud.ir/fa/databases/create/> and <https://docs.arvancloud.ir/fa/databases/> (mirror fetched 2026-08-26).
- Redis is available on **Cloud Container** (Kubernetes PaaS) via the Opstree Redis operator v0.17.0 (image `quay.io/opstree/redis:v7.0.12`): standalone node or cluster (3 leaders + 3 followers), applied with `kubectl apply`. Source: <https://docs.arvancloud.ir/fa/cloud-container/operators/redis/> and `.../cloud-container/catalogs/redis/`. You operate it yourself; it is not a managed DBaaS.
- Option for filetransfer.ir: run Redis on your own compute VM, or on Cloud Container, or use a hosted Redis elsewhere.

---

## Items still open (need Arvan sales/support or a paid test)

1. Which overage rate (3,000 tiered vs 1,400 flat) applies to Growth/Professional storage overage (Q1).
2. How CDN-fronted bucket traffic is metered (Q2), and what "دریافتی" means in CDN fair usage.
3. How multipart `UploadPart` is counted (Q3); also max object size and max part count (never found in the mirror; search terms "حداکثر حجم آبجکت", "حداکثر تعداد بخش", "5 TB", "10000 parts").
4. Range requests, edge signed URLs, presign max expiry and SigV4 (Q6).
5. Whether instant scan blocks download until done (Q8).
6. eNAMAD / licence duties for a public file host, and any hosting-side takedown process beyond the AUP (Q11).

## Provenance

- Live pages rendered with a browser on 2026-10-22: `/fa/pricing/cloud-storage`, `/fa/pricing/cdn`, `/fa/legal/terms`, `/fa/legal/acceptable-use-policy`. Raw text saved locally in `~/.playwright-mcp/` (`arvan-terms.txt`, `arvan-cdn-pricing.txt`, `arvan-aup.txt`); not committed.
- Docs-site pages came from the vendored mirror in `docs/upstream/pages/` (last revendored 2026-08-26, so docs-side claims could be ~2 months stale). Re-run `./scripts/fetch-upstream-docs.sh` and diff before relying on them.
- Prices change: Arvan's terms (§"تغییرات قیمت") say changes apply to existing customers after notice. Re-fetch pricing before the plan is finalized.
