# RFC (outline): Rules and Terms of Service for filetransfer.ir

**Status:** OUTLINE, not yet open for input. **Depends on:** the pricing RFC (Discussion #17), because refund, wallet, expiry and deletion rules are the pricing promise written as contract text. **Not legal advice:** every legal claim below needs review by an Iranian lawyer before launch.

## §1 — Why this is its own RFC

1. Zibal requires a valid eNAMAD and a business license before it issues a gateway (verified 2026-10-25). Reviewers of those will read our published rules, so the documents are on the critical path to taking any payment.
2. We host arbitrary public files. We carry content-liability, takedown and abuse duties that Arvan, our provider, pushes down to us (Arvan terms, 2026-10-22: customer is responsible for end users' content; Arvan may suspend without prior notice and may refuse refunds for criminal content).
3. The pricing RFC makes promises (price sheet, zero-balance deletion after 7 days, one-time free credit). Those must be consistent word for word with the terms.

## §2 — Documents in scope (Persian first, English optional)

| # | Document | Purpose |
|---|---|---|
| D1 | Terms of Service | The contract: account, service, price, wallet, termination |
| D2 | Acceptable Use Policy | What may not be uploaded or shared |
| D3 | Privacy Policy | What we collect and why; retention; deletion rights |
| D4 | Refund and wallet rules | Top-up, reservation, settlement, refund, expiry |
| D5 | Copyright and illegal-content notice, takedown and appeal | The process Arvan's pages lack |
| D6 | Supplier identity and contact page | E-Commerce Law identity disclosure |
| D7 | Price sheet page | One-minute pricing; generated from the same config as billing |

## §3 — Decision points (draft)

| Q | Decision | Options | Lean |
|---|---|---|---|
| T1 | Who is the contracting party? | Individual (حقیقی) vs company | Depends on the open question about the eNAMAD/license/tax registration; blocks D1 and D6 |
| T2 | Public file listing? | Links only, unlisted by default vs searchable | Links only. Smaller moderation surface |
| T3 | Content scanning | None at launch (antivirus is 11,000 toman/GB per engine) vs hash-block list vs on-report scan | No scanning; report-driven review plus a hash block list after a takedown |
| T4 | Takedown process | Written notice form with response deadline, counter-notice, repeat-offender rule | Define it ourselves; Arvan has none to inherit |
| T5 | Zero-balance deletion | Immediate link stop; files deleted after 7 days | Matches the pricing RFC Q5; terms must state it in plain words and notify by SMS |
| T6 | Refunds | Unused wallet balance refundable to the source card; consumed usage not refundable; refund fee borne by whom | Refund rules depend on Zibal corporate-banking refunds being available (otherwise manual) |
| T7 | Retention after deletion | Deleted data offline and separate per the privacy instruction vs hard delete | Needs lawyer input; object deletion must be tested |
| T8 | Data we collect | Phone number (OTP) only vs more | Phone only; minimise (privacy instruction) |
| T9 | Free-tier terms | One credit per verified phone; right to revoke for abuse | Matches pricing RFC Q4 |
| T10 | Government and legal requests | Policy for law-enforcement requests and legal holds | Needs lawyer input; Arvan already reserves the right to hand over information |
| T11 | Dispute handling and governing law | Iranian law; complaint address; response time | Standard; confirm with lawyer |
| T12 | Language | Persian authoritative; English courtesy translation | Persian authoritative |

## §4 — Verified inputs so far

- E-Commerce Law duties: supplier identity and contact, total charges, payment/delivery/cancellation terms, durable presentation, consumer cancellation rules (`docs/research/2026-10-22-business-compliance.md`, from WIPO Lex and law-ir.ir; secondary reading, not legal advice).
- 1402 privacy instruction: disclose what and why, required vs optional data, explicit consent, minimisation, deletion on request subject to legal retention (same doc, source nezamat.ir).
- Arvan terms and AUP (2026-10-22): no notice-and-takedown for third-party copyright; customer responsible for end-user content; prohibited categories under Iranian law; free tier has no SLA; no backups on their side.
- Zibal registration prerequisites (2026-10-25): eNAMAD, business license (mojavez.ir), tax file, identity documents.

## §5 — NOT yet verified (must research before synthesis)

1. Exact eNAMAD criteria for a file-hosting service, including whether a published rules page is checked.
2. Licensing duties for a public file host in Iran (the research doc says Arvan's pages do not answer this).
3. Whether individuals can operate this legally, or a company is needed.
4. Data-retention minimums for payment records and logs.
5. Whether the terms must be signed or only accepted by checkbox (E-Commerce Law reading).

## §6 — Plan shape (not started)

1. Open this as a Discussion after the pricing RFC synthesis lands.
2. Draft D1 to D7 in Persian as markdown under `docs/legal/`, rendered by the web app.
3. Product hooks needed (separate issues): consent version table and checkbox at signup, report-abuse form, admin takedown action, deletion job, SMS notice before deletion.
4. Lawyer review gate before any paid public launch (listed as a launch gate in `docs/research/2026-10-22-business-compliance.md`).

## §7 — How to respond

Comment on T1 to T12. T1 and §5 items 1 to 3 are the blockers.
