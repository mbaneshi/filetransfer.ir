# Arvan Cloud: findings from the arvancloud-eco knowledge base

Source: the `arv` session, answering from `~/arvancloud-eco` (mirrored fa docs, fetched 2026-08-26). Anything price-like must be rechecked against live pages before launch. A live pricing/terms pass (Q1-3, 6, 8, 11) is in progress and will land in a follow-up note.

Status key: CONFIRMED (a doc page says it), PARTIAL, NOT IN KB.

## Object storage

| Topic | Status | Finding | Affects |
|---|---|---|---|
| Multipart upload | CONFIRMED | Full API documented: initiate, upload-part, list-parts, list-multipart-uploads, complete, abort. Abandoned uploads MUST be aborted or the stored parts keep consuming (and billing) space. | #1 |
| Lifecycle rules | CONFIRMED | Object expiry with prefix/tag/size filters; incomplete multipart uploads are a listed use case. | #1 (free-tier expiry, cleanup) |
| Max object size / max part count | NOT IN KB | Test a real 50 GB upload before committing to the limit. | #1 |
| Presigned URLs | PARTIAL | Download and upload presign pages exist, using stock AWS SDKs. Examples use 1 h to 24 h expiry. SigV4 vs V2, max expiry, Range and Content-Disposition behaviour are not stated. | #1 |
| CORS | CONFIRMED | Configurable per bucket, including preflight handling. | #1 |
| Bucket limits | CONFIRMED | 50 buckets per account and 3,700,000 objects per bucket by default; both raisable via support. Buckets are region-bound. Bucket and object ACLs are independent (objects do not inherit). | #1, #4 |
| Per-user usage reporting | NOT IN KB | A Prometheus metrics endpoint (60 s) and a Grafana dashboard (#20106) exist, but nothing per user. Meter ourselves from presign/complete events; use access logs only as a cross-check. | #4 |
| Storage rate (3,000 vs 1,400 toman/GB) | NOT IN KB | Likely PAYG vs bundle overage (a guess). | #4 |
| Request pricing, multipart part counting | NOT IN KB | Docs say there is no request or traffic limit; nothing on billing. | #4 |
| Endpoints | CONFIRMED, **superseded 2026-10-23** | `ir-thr-at1`, `ir-tbz-sh1`. The panel now shows `s3.ir-central1.arvanstorage.ir` for a new Tehran bucket; an anonymous request's ID shows the same `ir-thr-at1` cluster behind it. The SigV4 signing region for the new name is unverified. See `docs/superpowers/specs/2026-10-23-deploy-arvan-box.md`. | infra |

## CDN

- Buckets are CDN-integrated by default; cache TTL is settable.
- For caching to work, use virtual-host-style URLs (`bucket.s3.REGION.arvanstorage.ir`). With a custom domain, the CDN domain's cache settings override the bucket's.
- Bucket "acceleration" (cache, firewall rules, image optimisation) is Growth/Pro/Enterprise only. Firewall rules can gate on cookies/headers, a possible token gate.
- NOT IN KB: max cacheable size, Range behaviour, edge signed URLs, whether CDN egress bills differently from direct S3 egress.

## Antivirus

- Engines: eScan, F-Secure, ESET, Kaspersky, Padvish. Growth plan and above only.
- Scan is manual or automatic per bucket; per-object status, CSV report, re-queue for unscanned files.
- NOT IN KB: whether auto-scan blocks download until the scan finishes; the 10,000 toman/GB price; any compliance requirement.

## Compute and hosting

- Compute is paid from the first hour; no free tier.
- VMs bill INBOUND traffic (250 GB free, then 1,800 toman/GB, source dated 2026-08-03). Uploads through our own VM would count; presigned direct-to-S3 uploads avoid that meter. This supports the direct-upload design in #1.
- Managed Postgres/MySQL exist (Terraform dbaas resources). Managed Redis: not confirmed.
- IaaS outbound traffic is SNAT'd (shared, rotating IP); there is no static egress IP product. This matters only if Zibal or Kavenegar need an allowlisted server IP.

## DNS and TLS

- Arvan DNS and CDN serve `.ir` zones.
- Trap 1: new A records default to CDN ON, which breaks the origin's own ACME issuance (Caddy). Turn CDN off for those records, or use DNS-01.
- Trap 2: the Arvan DNS module for Caddy is not on the official caddyserver.com download server; build with `xcaddy` and smoke-test it.
- The DNS panel zone dropdown does not reset between edits; check the zone before adding a record.

## Open (research pass pending)

Storage rate, CDN egress billing, multipart request counting, CDN behaviour for ~50 GB files, antivirus price and blocking behaviour, ToS/AUP/takedown/KYC for a public file host, eNAMAD requirements, managed Redis, contract and billing terms.
