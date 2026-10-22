# filetransfer.ir: metering and pricing model (on Arvan Object Storage)

> Written 2026-10-22 by the `arv` session. Desk work only: every Arvan number comes from this
> repo (`filetransfer-ir-arvan-questions-2026-10-22.md` and the mirrored docs). Nobody at Arvan
> was contacted and nothing in `~/filetransfer.ir` was changed.
> Owner rules: prices SIMPLE and TRANSPARENT (one-minute price sheet, no per-request fees shown
> to users, no surprise bills); internal rule customer price = Arvan cost x 1.20.
> Tags: **CONFIRMED** (a fetched page states it, and does not contradict itself),
> **CONTRADICTED** (two statements on Arvan's pages disagree), **UNKNOWN** (not in the KB),
> **ASSUMED** (my placeholder, not an Arvan fact; every one is listed in section 0.3).
> Where a number is CONTRADICTED or UNKNOWN I use the higher-cost reading.

---

## 0. Inputs

### 0.1 Number register (Arvan)

Sources: **P-ST** = <https://www.arvancloud.ir/fa/pricing/cloud-storage> (fetched 2026-10-22);
**P-CDN** = <https://www.arvancloud.ir/fa/pricing/cdn> (2026-10-22);
**TERMS** = <https://www.arvancloud.ir/fa/legal/terms> (2026-10-22);
**DOCS** = <https://docs.arvancloud.ir/fa/...> via the local mirror `docs/upstream/pages/` (2026-08-26);
**P-VM** = <https://www.arvancloud.ir/fa/pricing/cloud-server> as recorded in
`docs/reference/research/arvancloud-compute-gpu-pricing.md` (2026-08-03, **stale**, not refetched).

| # | Number | Value (toman) | Source | Status |
|---|---|---|---|---|
| N1 | Storage, tiered PAYG, 5 GB-5 TB, per GB | 3,000 (then 2,600 / 2,300 / 2,000 at 5 / 50 / 500 TB) | P-ST, "مصرف مازاد" table | **CONTRADICTED** by N2 |
| N2 | Storage overage in FAQ, flat per GB | 1,400 | P-ST, FAQ "منظور از مصرف مازاد…" | **CONTRADICTED** by N1 |
| N3 | Free storage (Base package) | 5 GB | P-ST | CONFIRMED |
| N4 | Download traffic, 20 GB-20 TB, per GB | 850 (700 / 600 / 500 at 20 TB / 200 TB / 2 PB) | P-ST (table and FAQ agree) | CONFIRMED |
| N5 | Free download (Base) | 20 GB/month | P-ST | CONFIRMED |
| N6 | Read requests per 10,000 | 500 (400 / 350 / 300 at higher volume) | P-ST | CONFIRMED |
| N7 | Write requests per 10,000 | 6,000 (Pro/Ent shown as 6,100) | P-ST | CONFIRMED |
| N8 | Included read+write requests | Base 10,000; Growth 20,000; Pro 50,000; Ent 100,000 | P-ST matrix | CONFIRMED (the overage table shows Pro reads free to 100,000: **CONTRADICTED**, immaterial) |
| N9 | Does an S3 `UploadPart` count as a write? | not stated | P-ST, DOCS multipart page | **UNKNOWN** (assume yes) |
| N10 | Max object size, max part count | not stated | searched DOCS | **UNKNOWN** |
| N11 | CDN overage traffic, per GB | 1,500 beyond fair usage; fair usage Base 8 / Growth 50 / Pro 500 GB per day | P-CDN FAQ | CONFIRMED for CDN; whether bucket traffic hits this meter: **UNKNOWN** |
| N12 | Antivirus, per GB scanned, per engine | 10,000 (add-on, Growth and up) | P-ST; DOCS antivirus page | CONFIRMED |
| N13 | VM inbound traffic | 250 GB/account/month free, then 1,800 per GB | P-VM (2026-08-03) | CONFIRMED but stale |
| N14 | Growth package / Pro package per month | 1,500,000 (500 GB, 2 TB) / 15,000,000 (5 TB, 20 TB) | P-ST | CONFIRMED |
| N15 | Wallet needed to buy a package | 50 % of package price (14 days) up front | P-ST FAQ | CONFIRMED |
| N16 | Package changes | at most 2 per month | P-ST FAQ | CONFIRMED |
| N17 | Unused included quota carries over? | not stated | P-ST, TERMS | **UNKNOWN** (assume no rollover) |
| N18 | Bucket limits | 50 buckets/account; 3,700,000 objects/bucket default (raisable) | DOCS `object-storage/limits/` | CONFIRMED |
| N19 | Max cacheable file in CDN | 100 MB default; Growth 256 MB; Pro 512 MB; Ent 2,048 MB | DOCS `cdn/caching/` | CONFIRMED |
| N20 | Upload (ingress) traffic charge | no line for it on P-ST | P-ST | **UNKNOWN** (absence; assume free) |
| N21 | Storage billing granularity (hourly / daily / monthly peak) and GB vs GiB | not stated | P-ST | **UNKNOWN** |
| N22 | Does Arvan VAT apply on top of listed prices? | TERMS say the final price "after VAT" appears on the invoice | TERMS "مالیات بر ارزش‌افزوده" | **UNKNOWN** rate and inclusion |
| N23 | Abandoned multipart parts keep consuming (billed) storage until aborted | stated | DOCS multipart page | CONFIRMED |
| N24 | Lifecycle can expire objects and incomplete uploads; minimum granularity | feature stated; min days not | DOCS lifecycle page | CONFIRMED / min days **UNKNOWN** |
| N25 | Bucket metrics export (Prometheus, 60 s) | stated | `arvancloud-cicd-observability-managed-services.md` §3.3b | CONFIRMED (aggregate per bucket, not per user) |

### 0.2 Scenarios used everywhere below

Storage and traffic are the two contested meters, so I carry three cost scenarios:

| Scenario | Storage per GB-month | Download per GB | Meaning |
|---|---|---|---|
| **S1 optimistic** | 1,400 (N2 true) | 850 | FAQ is right; presigned downloads hit only the storage meter |
| **S2 documented** | 3,000 (N1 true) | 850 | Table is right; no CDN charge |
| **S3 conservative** | 3,000 | 2,350 = 850 + 1,500 | Table is right **and** bucket traffic is billed on both the storage and CDN meters (N11 applies) |

Every figure is then **all-in = list x 1.10**, a hedge for N22 (ASSUMED 10 % VAT on Arvan's side).

| Unit cost (all-in) | S1 | S2 | S3 |
|---|---|---|---|
| Storage per GB-month | 1,540 | 3,300 | 3,300 |
| Storage per GB-day (/30) | 51 | 110 | 110 |
| Download per GB | 935 | 935 | 2,585 |
| Write request each (6,000/10k x 1.1) | 0.66 | 0.66 | 0.66 |
| Read request each (500/10k x 1.1) | 0.055 | 0.055 | 0.055 |

**Which one to assume for safety: S3 with all-in VAT.** The CDN question matters because Arvan's
limits page says buckets are CDN-integrated by default (<https://docs.arvancloud.ir/fa/object-storage/limits/>, mirror fetched 2026-08-26), so a presigned URL on
the bucket hostname may already be CDN traffic. The gap between S2 and S3 is 2.76x on the line
that dominates our margin, so we price on S3 until a test shows otherwise (section 3).

### 0.3 My placeholders (ASSUMED, not from Arvan)

| Item | Value | Why |
|---|---|---|
| Arvan VAT hedge | x 1.10 | N22 UNKNOWN; verify before launch |
| Payment gateway fee | 2 % of top-ups | placeholder; no gateway fee is in this KB |
| Abuse / refund / fixed-cost cushion | 8 % | judgement |
| Total reserve r | 10 % of price | 2 % + 8 % |
| Part size for multipart | 8 MB | typical; sets request counts |
| Unit | GB = 10^9 bytes | conservative: smaller unit, more billed units per byte (N21) |

---

## 1. Cost model

### 1.1 Formulas

Let `S` = GB-months stored, `D` = GB downloaded, `W` = write requests, `R` = read requests,
`A` = GB-months of abandoned multipart parts, `V` = GB scanned by antivirus, `E` = engines.

```
storage      = S x C_s                       C_s = 1,540 (S1) or 3,300 (S2, S3)
download     = D x C_d                       C_d = 935 (S1, S2) or 2,585 (S3)
requests     = W x 0.66 + R x 0.055
parts        = ceil(size_MB / 8) + 2         writes for one multipart upload (N9 assumed)
reads        = ceil(size_MB / 8)             per ranged download (8 MB chunks, assumed)
abandoned    = A x C_s                       parts live until aborted (N23)
antivirus    = V x E x 11,000                10,000 x 1.1 (N12); per scan, rescans pay again
```

Request counts per size: 1 GB = 125 parts, 127 writes; 5 GB = 625 parts, 627 writes; 10 GB =
1,250 parts, 1,252 writes; 50 GB = 6,250 parts, 6,252 writes.

### 1.2 Request cost is small. Two checks.

| Case | Writes | Reads | Cost (all-in) |
|---|---|---|---|
| Upload 1 GB, download once | 127 | 125 | 127 x 0.66 + 125 x 0.055 = 84 + 7 = **91** |
| Upload 50 GB, download once | 6,252 | 6,250 | 4,126 + 344 = **4,470** |
| Tiny file (1 write + 1 read) | 1 | 1 | **0.72** |
| 10 million tiny files | 10M | 10M | 7.2M, but also hits the 3.7M-object bucket cap (N18) |

So requests stay under about 2 % of the price on any real file. Absorb them, and bill every file
as at least 1 MB so tiny-file spam pays (section 3).

### 1.3 Storage and download side by side

| Per GB | S1 | S2 | S3 |
|---|---|---|---|
| Store 1 month | 1,540 | 3,300 | 3,300 |
| Download once | 935 | 935 | 2,585 |
| Store 1 month + download once | 2,475 | 4,235 | 5,885 |

Volume discounts exist (N4: 700 above 20 TB/month, 600 above 200 TB) but are ignored except in
the download-bomb example. We start small; do not price on them.

### 1.4 Buy a package or stay PAYG?

Growth is 1,500,000 per month for 500 GB storage + 2 TB traffic (N14). At S2 list prices, filling
it by PAYG would cost 3,000 x 495 + 850 x 1,980 = 1,485,000 + 1,683,000 = **3,168,000**, so a
full Growth package saves 1,668,000 (53 %). Break-even: switch when the PAYG bill would exceed
1,500,000 toman/month while staying under 500 GB / 2 TB (e.g. 250 GB stored + 1.2 TB served:
735,000 + 1,003,000 = 1,738,000 > 1,500,000; 200 GB + 1 TB: 585,000 + 833,000 = 1,418,000 < 1,500,000).
The 50 % up-front rule (750,000 wallet to buy, N15) and "2 changes per month" (N16) make this a
planned move, not a daily one. Feature gating on Base is UNKNOWN: the plan matrix marks lifecycle,
bucket policy and versioning with icons that the text capture could not read, so verify that lifecycle
works on Base before launch.

---

## 2. Hidden-cost audit (ranked by risk to margin)

Exposure uses the S3 all-in costs unless stated.

| Rank | Hidden cost | How a user action causes it | Exposure | Guard |
|---|---|---|---|---|
| 1 | **Download bombs and leaked presigned links** | A presigned URL is a bearer link: anyone holding it can fetch the file until it expires, with no call to our API. A 5 GB file, 15-minute link, 200 parallel connections at 12.5 MB/s: 200 x 12.5 MB/s x 900 s = 2,250 GB moved against **one** 17,500 charge. | 2,000 GB x 2,585 = **5.2M toman** for one link; a loop of fresh presigns is unbounded | Reserve the full file price at presign time; expiry <= size / 5 MB/s (min 5 min); per-file concurrency and daily GB caps; per-IP rate limits; Prometheus tripwire on bucket egress (N25) that flips the bucket policy to deny |
| 2 | **Meter ambiguity (CDN + storage)** | Every GB downloaded may cost 2,585 instead of 935 | +1,650 per GB, 2.76x. On 1 TB/month: +1.65M toman | Price on S3 until measured; run the one-account metering test in section 3.3 |
| 3 | **VM inbound if uploads or reads pass through a server** | A resumable-upload server such as tusd that streams bytes through our VM turns every uploaded GB into **inbound VM traffic: 1,800 per GB beyond 250 GB/month** (N13, stale). That is higher than the storage rate. | A 50 GB upload through the VM = 90,000 toman beyond the free 250 GB; 10 such uploads a month = 900,000 | Keep uploads browser to bucket with presigned multipart; if tusd stays, run it so data goes direct, and confirm whether intra-datacenter bucket traffic counts as VM inbound (UNKNOWN) |
| 4 | **VAT and billing granularity** | If listed prices exclude VAT (N22) or storage is billed on the monthly peak (N21), real cost rises | +10 % on everything; peak billing adds churn: a file stored 1 day may cost a full GB-month (3,300) against a 147 daily charge | The 1.10 hedge; charge per started day, not per hour; measure on first invoice |
| 5 | **Antivirus** | Scan on upload of a large file | 50 GB x 11,000 = **550,000** per engine per scan, more than the 395,000 revenue in worked example 2 | Do not offer scanning at launch (decision D3); if required, cap scanned size and price it explicitly |
| 6 | **Abandoned multipart parts** | A client starts an upload, sends 49 GB of parts, vanishes. Parts keep billing until aborted (N23). | 49 GB x 3,300 x 7/30 = **37,730** per abandoned upload held for a week; 1,000 bots = 37.7M | Lifecycle rule to abort incomplete uploads after 1 day (min days UNKNOWN, N24); per-account cap on open upload bytes; abort on our side after 24 h |
| 7 | **Free-tier farming and retention after wallet runs out** | OTP-only signup is cheap to farm; unpaid files linger | A 10,000-toman credit costs us up to 7,500 (worked example 4); unpaid 50 GB held 30 days = 165,000 | One-time credit only; 7-day grace then delete; cap files per free account |
| 8 | **CDN fair-usage overage** | Only if bucket traffic hits the CDN meter: beyond 8 / 50 / 500 GB per day (Base / Growth / Pro), 1,500 per GB (N11) | A 200 GB day on Base: (200 - 8) x 1,500 = 288,000 | Covered in S3; the direction of "received traffic" is ambiguous (UNKNOWN) |
| 9 | **Request costs on tiny files** | Millions of 1-byte files | 0.72 per file, plus the 3.7M-object bucket cap (N18) | 1 MB minimum billable size; bucket sharding plan |
| 10 | **Package rules** | Package buy needs 750,000 up-front (N15), two changes per month (N16), unused quota probably does not roll over (N17 UNKNOWN) | A Growth package bought for a quiet month wastes up to 1.5M | Buy Growth only at the break-even in 1.4 |

---

## 3. Proposed public price sheet

### 3.1 The three numbers (launch-safe, built on S3)

Price = all-in cost x 1.20 / (1 - r) with r = 10 %, i.e. all-in cost x 1.333. This leaves exactly the
owner's 20 % markup after the 10 % reserve in the worst case (S3).

| User-facing | Price | Derivation (S3 all-in) |
|---|---|---|
| **Storage** | **4,400 toman per GB per month** (charged per day, 147/GB/day) | 3,300 x 1.333 = 4,400 |
| **Download** | **3,500 toman per GB** | 2,585 x 1.333 = 3,447, rounded up to 3,500 |
| **Free credit** | **10,000 toman, once, at signup** | caps our free-tier loss at about 7,500 per account |

One-minute explanation for users: "You pay for the space your files use and for each GB that people
download. Uploading is free. Nothing else is charged. If your balance reaches zero, links stop and
files are deleted after 7 days."

**Deliberately not charged:** uploads (ingress is not billed, N20), API calls, requests, multipart parts,
link creation. We absorb them: 91 toman for a 1 GB upload + download (section 1.2) is about 1 % of the price
and sits inside the 10 % reserve. Internal rules, not user-facing numbers: every file is billed as
at least 1 MB; download is reserved from the wallet when the link is issued; a link's price is the
file size, whether or not the recipient finishes.

### 3.2 Sheets by verified cost

Because price cuts are easy and price rises hurt trust, launch on the safe column and cut once Arvan's
bill proves lower costs.

| Verified cost reality | Storage / GB-month | Download / GB |
|---|---|---|
| S3 (storage 3,000, CDN double meter) | 4,400 | **3,500** |
| S2 (storage 3,000, downloads 850 only) | 4,400 | **1,250** (935 x 1.333) |
| S1 (storage 1,400, downloads 850 only) | **2,100** (1,540 x 1.333 = 2,053) | 1,250 |

### 3.3 One cheap test that decides the sheet

Buy the Base (free) account's headroom plus a small wallet, upload 1 GB, download it 100 times via
presigned URLs, and read the wallet deduction. Per GB: about 935 means S2, about 2,585 means S3.
That costs up to about 260,000 toman and answers rank-2 risk directly. Run it before publishing.

### 3.4 Alternative design: flat plans

| | PAYG (3 numbers) | Flat plan |
|---|---|---|
| Example | 4,400 / 3,500 / 10,000 | 100 GB storage + 200 GB download per month for 1,150,000 |
| Derivation | per-unit | S3 cost: 100 x 3,300 + 200 x 2,585 = 847,000 -> x 1.333 = 1,129,000 -> 1,150,000 |
| User understanding | pay for what you use | one price, two caps |
| Surprise bills | none (prepaid, stops at zero) | none (hard caps) |
| Margin risk | cost-plus on every unit; risk is only the meter we mis-estimate | bounded: worst case is a user at both caps (S3 margin (1,150,000 - 847,000) / 1,150,000 = 26 % gross, 16 % after reserve). Light users are pure profit and may feel overcharged |
| Fit with prepaid wallet | native | needs a monthly subscription flow |

**Verdict:** PAYG is simpler for sharing (no plan to outgrow) and exact about cost. Flat plans win
later, because once we buy an Arvan Growth package, 20 % of it (100 GB + 400 GB, pooled cost 300,000,
all-in 330,000) can be resold for about 440,000. That is 2.6x cheaper than the PAYG-derived plan
above, thanks to the 2 TB of included traffic, but only if we fill the package (fixed 1.5M/month).
So: launch PAYG; add a plan after the package is bought and utilisation passes break-even.

---

## 4. Worked examples

Prices: storage 4,400 per GB-month, download 3,500 per GB. Costs are all-in (x 1.10) with the
request formulas of section 1. "Margin" = (price - cost) / price, before the 10 % reserve.
The uploader pays for downloads. Requests are counted at 8 MB parts.

### 4.1 A 1 GB file shared 100 times (stored one month)

Price = 4,400 + 100 x 3,500 = **354,400**. Requests: 127 x 0.66 + 100 x 125 x 0.055 = 84 + 688 = 771.

| | S1 | S2 | S3 |
|---|---|---|---|
| Cost | 1,540 + 93,500 + 771 = 95,811 | 3,300 + 93,500 + 771 = 97,571 | 3,300 + 258,500 + 771 = 262,571 |
| Margin | 258,589 (73.0 %) | 256,829 (72.5 %) | 91,829 (25.9 %); after the 10 % reserve (35,440) 56,389 (15.9 %) |

### 4.2 A 50 GB file downloaded once (stored one month)

Price = 50 x 4,400 + 50 x 3,500 = **395,000**. Requests: 6,252 x 0.66 + 6,250 x 0.055 = 4,126 + 344 = 4,470.

| | S1 | S2 | S3 |
|---|---|---|---|
| Cost | 77,000 + 46,750 + 4,470 = 128,220 | 165,000 + 46,750 + 4,470 = 216,220 | 165,000 + 129,250 + 4,470 = 298,720 |
| Margin | 266,780 (67.5 %) | 178,780 (45.3 %) | 96,280 (24.4 %) |

Add antivirus (one engine) and cost jumps by 550,000, giving margin **-453,720** in S3. Never auto-scan.

### 4.3 10 GB stored for a year, no downloads

Price = 10 x 4,400 x 12 = **528,000**. Requests: 1,252 x 0.66 = 826.

| | S1 | S2 / S3 |
|---|---|---|
| Cost | 184,800 + 826 = 185,626 | 396,000 + 826 = 396,826 |
| Margin | 342,374 (64.8 %) | 131,174 (24.8 %); after the 10 % reserve (52,800) 78,374 (14.8 %) |

If Arvan bills the monthly peak (N21) this stays valid only because the file is permanent; the peak
rule hurts churny short-lived files (rank 4).

### 4.4 A free-tier user at their limit (10,000 credit)

| Spent on | Consumption | Our cost S1 | Our cost S2 | Our cost S3 |
|---|---|---|---|---|
| Downloads only | 10,000 / 3,500 = 2.86 GB | 2,671 | 2,671 | 7,385 |
| Storage only | 10,000 / 4,400 = 2.27 GB-months | 3,500 | 7,500 | 7,500 |

Margin is **-100 %** by design (a marketing cost): we lose up to **7,500 per signup**. 1,000 signups =
7.5M toman. Guards: one credit per verified phone, a cap on credit signups per day, and the fact that
reservation at presign time means a free user cannot even download a file over 2.86 GB.

### 4.5 Download bomb: one 5 GB file downloaded 10,000 times

Traffic = 50,000 GB. Arvan download tiers (N4): first 20 GB free, then 850 up to 20 TB, then 700
to 200 TB.

| | S2 | S3 |
|---|---|---|
| Arvan cost (download only) | (19,980 x 850 + 30,000 x 700) x 1.10 = 41.78M | + 50,000 x 1,500 x 1.10 = 82.5M -> **124.28M** |
| Reads (10,000 x 625 x 0.055) | 0.34M | 0.34M |
| Total | **42.1M** | **124.6M** |
| Revenue if fully billed and paid | 175.0M | 175.0M |
| Margin | 132.9M (76 %) | 50.4M (28.8 %) |

Margin goes **negative** when the bomb is not billed, which happens in two ways:

| Failure | Result | Guard that stops it |
|---|---|---|
| Wallet debited after the fact, balance low | Loss = the whole cost, up to **124.6M** | Reserve 17,500 per presign up front; with a 100,000 balance only 5 downloads pass |
| One presigned URL reshared to many clients (rank 1) | One 17,500 charge, up to ~2,000 GB served = 5.2M (see section 2) | 5-minute expiry for large files, concurrency and per-IP caps, per-file daily GB cap, egress tripwire that denies the bucket |
| Free account bombs its own file | 5 GB reserve (17,500) exceeds the 10,000 credit, so nothing is served | reservation rule |

---

## 5. Metering spec (our own database, since Arvan has no per-user reporting)

Arvan gives only account-level wallet debits and per-bucket aggregates (N25). So our ledger is the
source of truth for user billing; Arvan's bill is the cost check.

### 5.1 Events (append-only table `usage_events`)

| Event | Fields (besides `event_id`, `ts_utc`, `user_id`, `file_id`) | Notes |
|---|---|---|
| `upload_init` | declared_size, part_size, expected_parts, bucket, key, client_ip_hash | creates the multipart upload |
| `part_presigned` | part_no, expires_at | one row per part URL issued |
| `upload_complete` | verified_size (from HEAD), etag, parts_count | size comes from Arvan, not the client |
| `upload_abort` / `upload_expired` | parts_bytes_estimated, reason | the abort call must succeed or be retried (N23) |
| `download_presigned` | bytes_reserved = size, expires_at, ip_hash, wallet_tx_id | the billable download event |
| `delete` | actor (user / expiry / moderation), size | stops storage accrual the same day |
| `expire` | policy (free_7d, wallet_zero_grace) | triggers delete |
| `wallet_tx` | kind (topup / reserve / settle / release / credit), amount_toman | double-entry |
| `daily_storage_snapshot` | sum bytes live per user at 00:00 +03:30 | derived from the DB, cross-checked against the bucket listing |

### 5.2 Rollups

- Hourly: bytes reserved for download, open upload bytes, per-bucket egress, to drive the tripwire.
- Daily close at 00:00 +03:30: storage accrual per user = live_bytes / 10^9 x (4,400 / 30), minimum 1 MB per file;
  post the charge to the wallet.
- Monthly: the reconciliation below.

### 5.3 Reconciliation with Arvan

Expected Arvan cost for the month, from our ledger:

```
expected = (GB-days stored /30) x C_s + GB reserved for download x C_d
           + (writes x 0.6 + reads x 0.05) + abandoned-part GB-months x C_s
```

Compare against the sum of Arvan wallet debits for the month. Report drift
`(actual - expected) / expected`:

| Drift | Action |
|---|---|
| within +/-3 % | fine |
| 3 % to 10 % | open an investigation, check abandoned parts and billing granularity |
| above 10 %, or above 100,000 toman absolute | freeze price changes, review the meter assumptions, check Prometheus bucket egress against our `download_presigned` totals |

Also keep a weekly cross-check of bucket total bytes (list the bucket) against the sum of our live
files, and bucket egress (Prometheus) against our reserved download bytes. A persistent gap larger than
5 % means leaked links or unmetered traffic.

---

## 6. Decisions for the owner

| # | Decision | Recommendation | The one number that would change it |
|---|---|---|---|
| D1 | Launch on S3 pricing or S2? | S3 (3,500 per GB download); cut after verification | Wallet deduction per downloaded GB in the 100-download test: about 935 -> cut download to 1,250; about 2,585 -> keep 3,500 |
| D2 | Who pays download? | The uploader; recipients never need an account | Share of files whose uploaders refuse to prepay (needs user testing) |
| D3 | Antivirus | Off at launch | A legal opinion that scanning is required; cost is 11,000 per GB per engine |
| D4 | Free credit | 10,000 once per verified phone | Cost of an SMS OTP (not in this KB): if farming costs an attacker less than 7,500, tighten |
| D5 | Markup | 1.20 on cost after a 10 % reserve (price x 1.333) | Real payment-gateway fee: if above about 3 %, raise the reserve |
| D6 | VAT on the user price | Quote VAT-inclusive prices | Tax advice; N22 also unresolved on Arvan's side |
| D7 | Package strategy | Stay on Base + PAYG; buy Growth at 1.5M monthly PAYG spend | Monthly PAYG bill crossing 1,500,000 with usage under 500 GB / 2 TB |
| D8 | Retention after wallet reaches zero | 7 days, then delete | Cost per unpaid GB-day (110): only matters if abuse is high |
| D9 | Minimum billable size | 1 MB per file | Real count of tiny files per user |

---

## 7. Open questions for Arvan support (blocking prices), ordered by how much they move the price

The existing draft is `filetransfer-ir-arvan-support-questions-2026-10-22.md`. Its numbers are kept;
this order replaces its order, and the **new** items (marked +) are missing from it.

| Order | Question | Price impact | In the draft? |
|---|---|---|---|
| 1 | Which overage rate: 3,000 or 1,400 per GB, and per package? | up to 2.1x on storage | yes, Q1 |
| 2 | Do bucket downloads (presigned, or via the CDN) bill the storage meter, the CDN meter, or both? | 2.76x on downloads | yes, Q2 |
| 3 | + Are listed prices ex-VAT or inclusive, and which VAT rate applies? | +10 % on everything | no |
| 4 | + Billing granularity of storage (hourly / daily / monthly peak) and is "GB" 10^9 or 2^30? | up to full-month charge for short files | no |
| 5 | + Is traffic between our VM and the bucket counted as VM inbound (1,800 per GB beyond 250 GB/month)? And is bucket upload (ingress) free? | 1,800 per GB if yes | no |
| 6 | Antivirus: does auto-scan block downloads until finished, is it per scan including rescans, and is there a size cap? | 550,000 per 50 GB scan | yes, Q5 |
| 7 | Multipart parts as write requests; max object size and part count | small money, but decides if 50 GB works | yes, Q3, Q4 |
| 8 | + Does unused included quota (Growth: 500 GB, 2 TB) roll over, and what happens on a package change mid-month? | up to 1.5M wasted | no |
| 9 | + Minimum days for "abort incomplete multipart uploads" in lifecycle; does Base include lifecycle? | abandoned-part exposure (rank 6) | no |
| 10 | + Access logs or per-prefix usage: any per-key metering we can use for reconciliation? | reconciliation quality | partly (Q8 of the draft is "per-user usage reporting") |
| 11 | Range requests, CDN signed URLs, presign max expiry | leak-bomb controls | yes, Q7 |
| 12 | Notice-and-takedown / content complaints | legal, not price | yes, Q6 |

---

## Provenance

- Pricing, CDN and terms pages: rendered in a browser on 2026-10-22 (the plain HTTP fetch only sees an
  interstitial). Docs pages are from the vendored mirror (revendored 2026-08-26) and may be about two months stale.
- All Arvan prices can change on notice (TERMS "تغییرات قیمت"). Re-fetch the pricing pages before launch.
- ASSUMED items (section 0.3) are mine and are not Arvan data.
