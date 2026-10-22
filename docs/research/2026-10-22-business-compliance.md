# FileTransfer.ir: business, Arvan economics, and compliance notes

Research date: 2026-10-22. Prices and regulatory interpretations must be rechecked before launch.

## Arvan Cloud input

Arvan's public Cloud Storage page currently lists these packages and included quotas:

- Base: free, 5 GB storage, 20 GB traffic, and 10,000 read/write requests.
- Growth: 1,500,000 toman/month, 500 GB storage, and 2 TB traffic.
- Professional: 15,000,000 toman/month, 5 TB storage, and 20 TB traffic.
- Enterprise: custom, with unlimited resources described on the page.

Arvan says excess usage is billed PAYG. The page currently lists standard-storage tiers beginning at 3,000 toman/GB for 5 GB–5 TB, download traffic at 850 toman/GB for 20 GB–20 TB, read requests from 500 toman per 10,000, and write requests from 6,000 toman per 10,000. It also lists access points at 90 toman/GB and antivirus at 10,000 toman/GB. The FAQ separately describes package overage examples of 1,400 toman/GB storage and 850 toman/GB traffic, so the exact effective rate must be confirmed in the Arvan account/contract before pricing is finalized.

Sources:

- [Arvan Cloud Storage pricing](https://www.arvancloud.ir/fa/pricing/cloud-storage)
- [Arvan Object Storage/S3 documentation](https://docs.arvancloud.ir/en/object-storage/s3)

Arvan's documentation says the S3-compatible workflow uses an access key and secret key from the cloud panel. The application should keep those credentials server-side and issue short-lived presigned upload/download URLs; it should never expose the master secret to browsers.

## Proposed economics

The requested rule is cost plus 20%. We should implement it as a transparent cost ledger, not as a single fixed price:

```text
customer price = (Arvan storage + download + requests + antivirus/access-point
                  + infrastructure + payment fees + support/risk reserve)
                 × 1.20
```

The 20% markup is not the same as a 20% gross margin. A cost of 100 and a sale of 120 gives a 16.67% gross margin before fixed costs, refunds, taxes, abuse, and bad debt. Therefore the service needs a minimum prepaid balance and/or a minimum monthly charge, especially for high-download users.

Recommended initial commercial model:

1. Free tier: small quota, strict abuse/rate limits, expiring links, no guarantee of long retention.
2. Prepaid wallet in toman: user buys credit; usage is reserved before expensive operations and settled from metered usage.
3. Paid plans: bundles that include storage and traffic, priced from the latest Arvan cost snapshot plus 20% and rounded to customer-friendly toman amounts.
4. Enterprise: negotiated commitment, SLA, retention, support, and possibly dedicated Arvan resources.

Usage must be metered separately for stored GB-hours/GB-month, uploaded bytes, downloaded bytes, object operations, antivirus scanning, and any CDN/access-point traffic. The ledger needs immutable usage entries, reservations, settlements, refunds, invoices, and an audit trail.

## Compliance and terms checklist

This is an engineering and product checklist, not legal advice. A qualified Iranian lawyer/accountant should validate the launch package.

The Electronic Commerce Law requires clear supplier identity/contact information, total charges, contract/payment/delivery/cancellation terms, durable presentation of information, and consumer cancellation rules for distance transactions. It also contains protections for private data messages and copyright.

Sources:

- [Electronic Commerce Law of Iran, WIPO Lex](https://www.wipo.int/wipolex/en/legislation/details/7711)
- [Iran Electronic Commerce Law text](https://law-ir.ir/law/cat/1656/)

The 1402 privacy-protection instruction says online platforms should clearly disclose what data they collect and why, distinguish required from optional data, obtain explicit consent, minimize collection, support deletion requests subject to legal retention, and encrypt identity/personal data. It also describes keeping legally required deleted data offline and separated from public networks.

Source:

- [Privacy and user-data protection instruction](https://nezamat.ir/post-44947/)

The service therefore needs, at minimum:

- Persian Terms of Service and Acceptable Use Policy.
- Privacy Policy and cookie/analytics disclosure.
- Copyright and illegal-content notice/takedown process.
- Abuse reporting, moderation, suspension, and appeal workflow.
- Data retention/deletion schedule and legal-hold process.
- Refund, wallet, expiry, and account-closure rules.
- Business identity, support address, payment receipts, and tax/accounting review.
- Security controls: encryption in transit, encrypted identity data, secret management, audit logs, backups, malware scanning, rate limiting, and incident response.

## Product architecture implied by the business model

The application should add:

- PostgreSQL: users, sessions, files, upload sessions, links, plans, wallet accounts, ledger entries, invoices, refunds, abuse cases, consent versions, and legal holds.
- Redis: OTPs, sessions, upload locks, rate limits, link throttling, and jobs.
- Arvan S3: private object storage, multipart/resumable uploads, server-side encryption where available, lifecycle deletion, and presigned URLs.
- Worker process: finalize uploads, scan files, calculate usage, expire links, settle reservations, send notifications, and retry provider failures.
- Admin dashboard: usage/cost/revenue, wallet adjustments, abuse review, takedown, refunds, and provider reconciliation.

## Launch gates

Do not launch paid public access until the following are verified: Arvan contract and current tariffs, payment-provider settlement/refund behavior, legal terms review, business/tax obligations, abuse escalation contact, backup/restore test, object-storage deletion test, and a cost simulation covering heavy downloads.
