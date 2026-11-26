# Design: prepaid plans and limits in the API (Stage A)

**Date:** 2026-11-26. **Tracks:** #20. **Follows:** `2026-11-25-prepaid-plans-and-legal-proposal.md`, sections 3 and 4.

## State before this change (read from the code and the live site)

- Migrations are idempotent SQL files run at every API start (`apps/api/migrations`). There is no migration tool and no down-migration.
- Tables: `users`, `files` (status `uploading`, `ready`, `deleted`; rows are never removed), `shares`.
- Any logged-in user may upload files of up to `MAX_FILE_SIZE_BYTES` (50 GB). Nothing counts storage or downloads.
- A download is a 302 redirect to a presigned bucket URL valid for one hour. The API never sees the bytes.
- A janitor aborts multipart uploads older than 24 hours.

## What this change adds

| Piece | Where | Behaviour |
|---|---|---|
| Plan catalogue | `internal/plans` | Basic, Max, Ultra with the numbers from the proposal. 30-day period, 3 grace days |
| `subscriptions` table | `0003_plans.sql` | One row per purchased or hand-activated period |
| `downloads` table | `0003_plans.sql` | One row per download handed out; the sum is the traffic used |
| `files.expires_at` | `0003_plans.sql` | Retention date, set at upload from the plan |
| Upload gate | `POST /api/v1/uploads` | 402 `no_active_plan`, 413 `file_too_large_for_plan`, 409 `storage_full` |
| Share cap | `POST /api/v1/files/{id}/shares` | A link's expiry is cut to the file's retention date |
| Download gate | `GET /api/v1/shares/{token}/download` | 403 `plan_expired` after the grace days, 403 `traffic_exhausted` when the quota is used |
| Share info | `GET /api/v1/shares/{token}` | New field `blocked` with the same reason, so the page can explain |
| Account | `GET /api/v1/account` | Plan, dates and usage for the logged-in user |
| Catalogue | `GET /api/v1/plans` | Public list of plans |
| Deletion job | janitor | Removes files past retention, and all files of an owner whose last plan ended more than 3 days ago |
| Owner command | `filetransfer plan show|activate|cancel <phone>` | Hand activation until online payment exists |
| Switch | `PLANS_ENFORCED` | Default on. `0` restores the old open behaviour without a new image |

## Decisions and their reasons

1. **Plans live in code, not in a table.** Three fixed plans change rarely and a price change must not rewrite what a user already bought. The web app has its own copy of the table (`apps/web/app/lib/plans.ts`); the two must be changed together.
2. **Storage used = finished files plus uploads in progress.** Otherwise a user could start many uploads at once and pass the quota. Two uploads started in the same instant can still slip past by one file; the check is not locked.
3. **Traffic is counted when the link is handed out, as the whole file size.** The bucket serves the bytes, so the API cannot measure them. A cancelled download still counts, and a presigned URL reused within its hour does not. Exact numbers need the provider's bill (Stage C).
4. **Traffic resets with each subscription row.** A renewal is a new row that starts when the previous one ends.
5. **Files of users who never had a plan are never deleted by the job.** This protects the files uploaded before plans existed. Those files have no retention date either.
6. **The job deletes the object but keeps the `files` and `downloads` rows.** If the law turns out to require keeping file contents for a period after a subscription ends (open question in the proposal, section 5.2), a purge delay must be added before public launch.
7. **No admin web page.** The owner runs a command inside the API container. This adds no new surface on the public site.
8. **Cancelling moves `expires_at` to now.** The grace days and the deletion job then follow the same path as a normal expiry.

## Not in this change

- The two SMS notices at expiry and before deletion. They need an approved SMS template that does not exist yet.
- Web UI: account panel, the Persian messages for the new error codes, the share-expiry picker.
- Upgrade with a pro-rated difference, online payment (Stage B).
- Disabling a file or a user for abuse.

## Rollout

1. Deploy. The migration adds two tables and one nullable column; it changes no existing row.
2. From that moment nobody can upload without a plan, including the owner. Activate a plan first:
   `docker exec ft-api /filetransfer plan activate <phone> ultra`
3. To back out without a redeploy, set `PLANS_ENFORCED=0` in `app.env` and restart the API container. The new tables stay and do no harm.

## Verification

- `go test ./...`: unit tests for every rule above against an in-memory repository.
- `TEST_DATABASE_URL=... go test ./internal/files -run TestPGPlanQueries`: the SQL against a real Postgres, including running every migration twice.
- The `plan` command was run by hand against a throwaway Postgres: show, activate, renew, refuse a plan switch, cancel, activate again.
- Not verified: the full flow against the real bucket and the live site.
