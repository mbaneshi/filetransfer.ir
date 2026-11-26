# Proposal: prepaid plans, pricing and the legal path to a public launch

**Date:** 2026-11-25. **Status:** PROPOSAL, waiting for the owner's yes or changes. **Tracks:** #20. **Related:** #3 (payments), #4 (metering), #18 (terms), Discussion #17 (pricing RFC).

**Not legal or tax advice.** Every legal and tax statement here is a working assumption from our own research notes. Items marked **VERIFY** must be confirmed by an Iranian accountant or lawyer, or by the body named, before we rely on them.

## 1. Where we are

The service has been live on `https://filetransfer.ir` since 2026-11-25 (`v0.1.2` at the time of writing). SMS login, small and large uploads (a 150 MB file in three parts), share links and downloads were tested end to end, and the downloaded files matched the originals byte for byte.

The live version has no plans, no limits and no payment. Anyone with a mobile number can upload files of up to 50 GB, and the owner pays the storage bill. Search engines are blocked. The site must not be announced until section 4, stage A is done.

## 2. Decisions already made by the owner (2026-11-25)

| Decision | Detail |
|---|---|
| Markup | Price = cost x 1.20 / (1 - 10% risk reserve), about 1.33x cost |
| No free tier | Every user prepays before uploading |
| Three fixed plans | Basic, Max, Ultra. Limits and retention are fixed per plan |
| Metering | Usage is measured per user and checked against the plan |

This replaces the pay-per-byte wallet described in #3 and #4, and removes the "one-time free credit" idea (T9 in the terms outline).

## 3. Pricing proposal

### 3.1 The plans

A plan is a prepaid package that lasts 30 days from the moment of payment.

| | Basic | Max | Ultra |
|---|---|---|---|
| Storage at any one time | 10 GB | 50 GB | 200 GB |
| Download traffic per 30 days | 20 GB | 100 GB | 400 GB |
| Largest single file | 5 GB | 20 GB | 50 GB |
| Retention (a file is deleted this long after upload) | 7 days | 14 days | 30 days |
| Worst-case provider cost (toman) | 47,000 | 235,000 | 940,000 |
| **Price, toman per 30 days, before VAT** | **65,000** | **315,000** | **1,255,000** |

### 3.2 How the price is calculated

```text
cost  = storage GB x 3,000  +  download GB x 850
price = cost x 1.20 / 0.90, rounded up to the next 5,000 toman
```

| Plan | Storage cost | Download cost | Cost | x 1.333 | Price |
|---|---|---|---|---|---|
| Basic | 10 x 3,000 = 30,000 | 20 x 850 = 17,000 | 47,000 | 62,667 | 65,000 |
| Max | 50 x 3,000 = 150,000 | 100 x 850 = 85,000 | 235,000 | 313,333 | 315,000 |
| Ultra | 200 x 3,000 = 600,000 | 400 x 850 = 340,000 | 940,000 | 1,253,333 | 1,255,000 |

What the formula assumes, and what it leaves out:

- The rates are Arvan's published pay-as-you-go rates, read on 2026-10-22 and 2026-10-25. Arvan's FAQ shows 1,400 toman per GB for storage instead of 3,000. We price on 3,000, the safer number. **VERIFY** with Arvan support or the first real bill; if the true rate is lower, the margin is higher and the prices can be cut later. Cutting a price is easy, raising one is not.
- Cost is the worst case: the user fills the whole storage quota for all 30 days and uses all the traffic. Most users will not, so the real margin is normally above 33%.
- Not in the cost line: request fees (small), the SMS for each login, the gateway fee (1%, capped at 20,000 toman, plus VAT on the fee), the server (shared with another product today). These are what the 10% reserve and the unused quota are for.
- VAT is not included. **VERIFY** with the accountant whether we must charge it and from when; if yes, it is added on top at checkout and shown separately.

### 3.3 Rules that go with the plans

| Rule | Proposal | Why |
|---|---|---|
| Storage full | New uploads are refused until the user deletes a file or upgrades | Simple, no surprise bill |
| Download traffic used up | Links show "traffic limit reached" until renewal or upgrade | Caps our cost exactly |
| Plan expires without renewal | Uploads stop at once; links keep working for 3 days; then files are deleted. One SMS at expiry, one a day before deletion | Gives the user a fair warning and stops our cost |
| Renewal | Manual: the user buys another 30 days. No automatic card charge | Iranian gateways do not offer card-on-file renewal the way foreign ones do (**VERIFY**) |
| Upgrade mid-period | Pay the difference for the remaining days; limits rise at once | Avoids refund work |
| Downgrade | Takes effect at the next renewal | Avoids refund work |
| Refund | Full refund within 24 hours if nothing was uploaded; otherwise none. Refunds are done by hand | Gateway refunds need corporate banking (research note, 2026-10-25) |
| Extra traffic or storage | Not sold at launch. The user upgrades instead | Keeps billing to one kind of purchase |

### 3.4 What I recommend, in one line

Launch with the table in 3.1 as it stands, priced on the 3,000 rate, without VAT until the accountant says otherwise, and review prices after the first full month of real bills.

## 4. Engineering stages

### Stage A: plans and limits, with manual activation (no outside dependency)

- [ ] `plans` and `subscriptions` tables; a user with no active plan cannot start an upload
- [ ] Storage quota and largest-file limit checked when an upload starts
- [ ] Download traffic counted per user per period; links stop when it is used up
- [ ] Retention job: delete the object and the row when a file passes its plan's retention
- [ ] Expiry job: stop uploads, 3-day grace, delete, with the two SMS notices
- [ ] Admin action to activate, extend or cancel a plan by hand
- [ ] Admin action to disable a file or a user (abuse)
- [ ] Account page: plan, days left, storage used, traffic used
- [ ] Pricing section on the landing page, generated from the same plan table the API enforces
- [ ] Landing page: remove the "password on the link" sentence or build the feature
- [ ] Share-expiry picker, bounded by the plan's retention
- [ ] Fix the stale `uploadProtocol: "tus"` in `GET /api/v1/config`
- [ ] Report-abuse link on share pages
- [ ] Legal pages rendered by the web app (section 5.3), and a consent checkbox at login that stores the accepted version
- [ ] `www.filetransfer.ir` record and redirect
- [ ] Uptime check with an alert; alert on low provider balance
- [ ] Off-box copy of the nightly database backup, and one restore test
- [ ] Test that deleting a file removes the object from the bucket
- [ ] Rate limits on upload and download, per account and per IP address
- [ ] Access logs copied off the box and kept for the period the law requires (section 5.2)
- [ ] Alert when storage or traffic spend passes a set amount; one abusive account can erase the margin
- [ ] Written policy for malware and illegal content, even if the review is manual at first
- [ ] Incident runbook: who is told, what is switched off, how users are informed

Outcome: a closed beta is possible. The owner activates plans by hand for invited users.

### Stage B: online payment (blocked on section 5.1)

- [ ] Gateway request, verify and callback flow; a payment activates a plan exactly once even if the callback repeats
- [ ] Failed or abandoned payments never activate a plan
- [ ] `payments` table, append-only
- [ ] Receipt after payment (and a formal invoice if the accountant requires one)
- [ ] Upgrade with pro-rated difference
- [ ] SMS reminder 3 days before expiry

Outcome: anyone can buy a plan. This is the public launch.

### Stage C: after launch

- [ ] One month of metered usage reconciled against the provider bill; adjust prices
- [ ] Set `FT_ROBOTS` to `index, follow`
- [ ] Link passwords, per-link download limit, download counter shown to the owner of the file

## 5. Legal proposal

### 5.1 Recommended path: register as an individual first

Start as an individual (shakhs-e haghighi), not a company. Reason: it is the shortest route to a payment gateway, it has no company registration cost, and it can be converted to a company later when revenue justifies it. This answers T1 in the terms outline. **VERIFY** with the accountant that an individual may operate a paid file-hosting service, and what the tax consequences are compared with a company.

Order matters, because each step is an input to the next:

| # | Step | Where | Needs | Note |
|---|---|---|---|---|
| 1 | Accountant conversation | An accountant who handles online businesses | This document | One meeting. Settles individual vs company, VAT, invoices, record keeping |
| 2 | Tax file | The national tax portal | National ID, a business address and postal code | The gateway and eNAMAD ask for the tax tracking code |
| 3 | Legal pages live on the site | Us (stage A) | Section 5.3 | eNAMAD reviewers read the site. **VERIFY** exactly which pages they check |
| 4 | Trade-law training certificate | The course named in the licence conditions | Enrolment by the applicant | A condition of the guild licence (section 5.5). It takes time, so start now |
| 5 | Business licence | mojavez.ir (national licences portal) | Steps 2 and 4; a police (Amaken) eligibility check on the applicant; an address that matches Iran Post | There is no "file hosting" category. Ask the guild union which category it accepts before applying (section 5.5). The eNAMAD panel states (read 2026-11-25) that eNAMAD is issued only after the required licences from this portal are in place, so this comes before eNAMAD |
| 6 | eNAMAD | enamad.ir | Steps 2, 3 and 5, domain ownership in the same name as the applicant | Domain ownership was verified on 2026-11-25. Because we sell plans, the eNAMAD type must be one that allows online sales (section 5.5) |
| 7 | Payment gateway account | Zibal (or another provider) | Steps 2, 5 and 6, bank account in the same name | The gateway form asks for the tax tracking code and the postal code registered on the same tax file. Compare one other provider before committing |
| 8 | eNAMAD logo and gateway live on the site | Us (stage B) | Step 7 | |

Step 2 is done: the owner already has a completed tax file as an individual (checked 2026-11-25). Step 3 depends on stage A. Nobody can promise a date for steps 5 to 7; plan for weeks, not days.

Steps 5 to 7 include checks on the applicant that are outside our control. Stage A and a closed beta do not depend on them.

### 5.2 Open legal questions for the lawyer

These come from section 5 of the terms outline and are still unanswered:

- [ ] Does a public file-hosting service need a licence beyond eNAMAD and the business licence?
- [ ] What must we do when someone reports a file as illegal or as a copyright violation, and how fast?
- [ ] What records (payments, logs, phone numbers) must we keep, and for how long?
- [ ] Does the Computer Crimes Law oblige a hosting provider to keep user data or files for a period after the subscription ends? An unverified note says possibly six months. If yes, it conflicts with automatic deletion in section 3, and must be settled before the retention job and the privacy page are designed
- [ ] How do we answer a request from the authorities for a user's data or files?
- [ ] Is acceptance by checkbox enough, or must the terms be accepted another way?
- [ ] What must the site show about the seller (name, address, phone) under the E-Commerce Law?

### 5.3 Documents to publish (Persian)

| # | Page | Content | Source of the rules |
|---|---|---|---|
| D1 | Terms of Service | Account, the three plans, payment, expiry and deletion, termination | Sections 3.1 and 3.3 of this document |
| D2 | Acceptable use | What may not be uploaded or shared | Arvan's own terms push this duty to us |
| D3 | Privacy | We collect the phone number only; why; how to ask for deletion | 1402 privacy instruction (research note) |
| D4 | Refund rules | Section 3.3, in plain words | E-Commerce Law cancellation rules. **VERIFY** the 24-hour rule against the consumer's legal right of withdrawal |
| D5 | Report and takedown | How to report a file, what we do, how to appeal | Our own process; the provider has none |
| D6 | About and contact | Who runs the service, address, phone, email | E-Commerce Law identity disclosure |
| D7 | Prices | The table in 3.1 | Generated from the plan table |

I will draft D1 to D7 in Persian from this proposal. The lawyer reviews them before stage B.

### 5.4 Accounts and other paperwork

- [ ] Own SMS template at the SMS provider, replacing the borrowed one. Request now; approval takes time
- [ ] Support email address and phone number to publish
- [ ] Confirm the domain at IRNIC is registered in the same name that will apply for eNAMAD
- [ ] A bank account in that same name for gateway settlement

### 5.5 What the licences portal says (read 2026-11-25)

A second research pass read the listings on the national licences portal and the eNAMAD guide. Each line says whether it was read from the official source (**CONFIRMED**) or is our own reading (**ASSUMED**).

| Finding | Status |
|---|---|
| The portal has no "file hosting" category. The route is a guild business licence, issued by a guild union | CONFIRMED |
| The closest categories are "offering one group of services in online shops" and "computer services over the internet". Which one the union accepts for file transfer is the union's call | ASSUMED |
| For the licence whose guide was read in full: issued within 7 working days after all checks pass, valid for 5 years | CONFIRMED |
| Cost in a large city: union membership up to 2,000,000 toman in the first year, plus a 600,000 toman issuing fee | CONFIRMED |
| Conditions checked online: a tax file with no tax debt, address and postal code matching Iran Post, military-service status, an office address backed by a deed or a formal lease, the police eligibility check, and a certificate from the trade-law training course | CONFIRMED |
| The address and postal code should be identical on the licence application, the tax file and Iran Post's records before applying | ASSUMED |
| The applicant accepts commitments: traceable user identity, a complaints process, no commercial use of users' private data, no prohibited content, access logs at every layer, reporting a breach to the cyber police | CONFIRMED |
| eNAMAD type: "micro business with a transaction ceiling" or the ordinary one-star type. The "no online sales" type does not fit | CONFIRMED |
| Pages reviewers expect: terms, privacy, contact (address, landline, email), about, prices, refunds, complaints, prohibited content | ASSUMED |

The commitments row is, in practice, the outline for D1 to D5 in section 5.3, and "access logs at every layer" is why log retention was added to stage A.

Still unknown: the eNAMAD fee, the full document list for the gateway, the route for registering as a company, and how long the eNAMAD, gateway and police checks take.

## 6. Checklist for the owner

Now, no dependency:

- [ ] Say yes to section 3.1, or change the numbers
- [ ] Say yes to section 3.3, or change the rules
- [ ] Say yes to "individual first" in 5.1, or say company
- [ ] Book the accountant (5.1 step 1) and take section 3 and 5 of this document
- [ ] Request the SMS template
- [ ] Choose the support email and phone number
- [ ] Check the name on the domain registration

- [ ] Enrol in the trade-law training course (5.1 step 4)
- [ ] Make the address and postal code identical on the tax file, Iran Post and the licence application
- [ ] Ask the guild union which licence category it accepts for a file-transfer service

After the accountant:

- [ ] Find a lawyer for the questions in 5.2 and the review of D1 to D7

In this order, each after the one before:

- [ ] Apply for the business licence
- [ ] Complete eNAMAD (needs the licence and the legal pages from stage A)
- [ ] Complete the gateway account
- [ ] Invite the first closed-beta users and activate their plans by hand. **VERIFY** with the accountant how to take and record payment from them before the gateway exists

## 7. Risks

| Risk | Effect | Answer |
|---|---|---|
| The paperwork takes months | No public launch | Stage A and the closed beta do not depend on it |
| The real storage rate is higher than 3,000 | Margin shrinks | Plans are priced on the highest published rate; reconcile after month one |
| No free tier means few sign-ups | Slow growth | Owner's decision. Can be revisited with a short paid trial later, without changing the design |
| A user uploads illegal content | Provider may suspend the whole account without notice | Report link, admin disable, takedown process, and identity by phone number from day one |
| Prices need to rise | Unhappy users | Prices are per 30 days with no auto-renewal, so a change only applies at the next purchase |
