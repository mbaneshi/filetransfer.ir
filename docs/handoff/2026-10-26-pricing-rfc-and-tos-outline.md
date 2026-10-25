# Handoff 2026-10-26: pricing RFC is open; terms RFC is outlined; deploy still blocked

Read this first. The older deploy handoff (`2026-10-23-deploy-blocked-on-access-key.md`) is still accurate for the box and Arvan steps.

## Decisions made this session

| Decision | Detail |
|---|---|
| Domain | Use `filetransfer.ir`, not `ft.webcad.ir`, as the product domain |
| Priority order | 1. elastic resource, 2. transparent pay-as-you-go pricing, 3. rules and terms of service |
| Elastic resource means (proposed, RFC Q7) | Customers pay only for bytes stored and bytes downloaded. Not autoscaling infrastructure. **Confirm in the RFC.** |
| Pricing is a business-paradigm decision | It gets its own RFC before any metering or wallet code |

## State

| Piece | State |
|---|---|
| Pricing RFC | Discussion #17 (https://github.com/mbaneshi/filetransfer.ir/discussions/17), Phases 1 and 2 done. Verification comment posted. Issues #3 and #4 link to it. |
| Phase 3 (external input) | **Waiting on the owner**: send the section 5 prompt of the RFC to an outside expert and paste the reply back |
| Phase 4 (synthesis) | Planned 2026-10-27, unless objections |
| Terms RFC | Outline merged with this handoff: `docs/superpowers/specs/2026-10-26-terms-of-service-rfc-outline.md`. Not yet a Discussion. Tracked in #18. |
| Domain `filetransfer.ir` | Nameservers delegated to Arvan (`g.ns.arvancdn.ir`, `p.ns.arvancdn.ir`). **No A record yet.** Does not resolve. |
| Arvan S3 access keys | Not created. `app.env` on the box still has `CHANGE_ME` for both keys. |
| Live site | None. `ft.webcad.ir` returns Traefik 404 / bad cert (nothing routed). |
| v0.1.0 | Released. Has **no billing, quota or metering**: only a per-file size cap and OTP rate limiting. Do not open public signup until #3 and #4 ship. |

## Facts verified this session (2026-10-25)

- Arvan storage pricing page unchanged since 2026-10-22: storage 3,000 toman/GB, download 850, writes 6,000 per 10k, reads 500 per 10k, antivirus 10,000. The FAQ still says 1,400/GB for package overage, which contradicts the 3,000 table. Unresolved: only Arvan support or a paid test settles it.
- Zibal (https://zibal.ir/pricing, read through a summariser, owner to confirm by eye): gateway fee 1% per transaction, cap 20,000 toman, +10% VAT on the fee, no monthly fee. Refund fee cap 50,000. Refunds need corporate banking.
- Zibal prerequisites to get a gateway: valid eNAMAD matching the applicant, business license (mojavez.ir), tax file in the applicant's name, identity documents. See https://help.zibal.ir/article/registration/.
- Contradiction inside the project: issue #4 says `(costs + reserve) x 1.20`; the research model computes `cost x 1.20 / (1 - 10% reserve)`, effectively x1.333. The RFC (Q1) asks which one is the public promise.

## Open questions only the owner can answer

1. Do eNAMAD, a business license and a tax file already exist? Individual (حقیقی) or company registration? This decides whether the wallet launch is days or weeks away, and blocks terms item T1.
2. Does "elastic resource" mean pay-only-for-bytes (recommended) or also autoscaling?
3. Q1 to Q6 of the pricing RFC.

## Next steps, in order

1. Owner: send the RFC section 5 prompt to an external expert, paste the reply. Claude verifies each load-bearing claim before use (Iranian e-commerce rules, Zibal, VAT).
2. Claude: post the Phase 4 synthesis on Discussion #17 and a decision table on #3 and #4.
3. Owner: create the Arvan S3 access-key pair and run the deploy steps in the 2026-10-23 handoff. After that, create the `filetransfer.ir` A record (proxy off first) to `85.198.52.103`.
4. Claude: open an RFC (`rfc-flow`) for the hostname change before touching prod routing. The CORS origin, Traefik host rule, cert and public URL config are all written for `ft.webcad.ir`.
5. After the pricing synthesis: open the terms RFC as a Discussion, then build #4 (metering) and #3 (wallet) per the converged outline.

## Housekeeping left

- `promote` script fails to fast-forward `main` when `main` is checked out in a worktree (`git branch -f`). Fix it.
- The startup hook says the repo is missing project notes and a decision ledger: run `repo-reconcile --apply .`, then `prostat propose` and let the owner lock decisions. `prostat brief` still shows no locked decisions.

## Permissions notes for the next session

- `gh` writes (issue comments, Discussion posts, PR create and merge) are denied for Claude here. Hand the owner the command with the `!` prefix. Reads work.
- Pass Discussion bodies with `-f body="$(cat file.md)"`, never `jq -Rs` (see `~/.claude/docs/gh-gotchas.md`).
- Playwright's file output only writes under the repo directory. Delete `.playwright-mcp/` after use.
- The repo is **public**: anything posted to Issues or Discussions is world-readable.
