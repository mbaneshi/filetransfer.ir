# Deploy filetransfer.ir to the Arvan box (RFC record)

Status: decided 2026-10-23, scripts merged (#13, #15), box bootstrapped, **first real deploy not yet run: blocked on the Arvan access-key pair** (see section 4 and `docs/handoff/2026-10-23-deploy-blocked-on-access-key.md`). Issue: #12.
Template: `nexus-platform-ir/deploy/webcad` (build on the Mac, ship a tarball, remote script verifies and rolls back).

## 1. Verified live state (2026-10-22 and 2026-10-23, read-only SSH to `85.198.52.103`)

| Fact | Source command | Consequence |
|---|---|---|
| 3.9GB RAM, ~2.5GB available, 2GB swap | `free -m` | Four small containers (~640MB of limits) fit. The box cannot build images. |
| 23GB disk, 12GB free | `df -h /` | Image tarballs are small; `ft-deploy` still refuses above 80% used. |
| Traefik v3.6 on Docker network `web`; entrypoints `web`/`websecure`; ACME resolver `le` (HTTP-01) | `docker inspect traefik` | We attach with labels only; no Traefik changes. |
| 13 containers already run (Supabase, webcad-app, landing, smsgate-bridge, sms-hook) | `docker ps` | Shared production box: nothing here may touch their networks, volumes or backups. |
| Nightly Supabase backup at 03:30 via `/opt/webcad/backup.sh` | `ls /opt/webcad/backups`, `/etc/cron.d/webcad-backup` | Ours is separate (`/etc/cron.d/filetransfer-backup`, 03:45). |
| Docker Hub and ghcr answer (401 unauthenticated) | `curl` from the box | One-time base-image pull at bootstrap works; deploys never need a registry. |
| `filetransfer.ir` is registered at IRNIC but has **no nameservers**, and is not in the Cloudflare account | `whois -h whois.nic.ir`, `dig`, infra `zone_status` | No public DNS or TLS for it yet. |
| `scripts/promote` exists; repo has 0 tags; `main` is the scaffold, and `origin/main` does not exist yet | `git tag`, `git branch -r`, `scripts/promote` | The first `promote` creates it (the script handles a first release). It refuses a dirty tree. |
| `webcad.ir` has a **wildcard** record: any undefined subdomain resolved to `185.206.93.107` (AbrArvan), not our box. An explicit `ft A 85.198.52.103` record overrides it (verified 2026-10-23 via `1.1.1.1` and `8.8.8.8`, TTL 120) | `dig`, `curl -I` | Without the explicit record, Let's Encrypt and the health check would hit the wrong server. Keep the Arvan proxy toggle OFF for this record so HTTP-01 reaches Traefik. |
| `webcad.ir` nameservers are Arvan's (`*.ns.arvancdn.ir`); the `infra` MCP only manages Cloudflare | `dig NS`, `zone_status` | DNS for this zone is edited in the Arvan panel by hand. |
| Bucket `filetransfer` exists (Standard, Iran Central 1, private). The panel shows its endpoint as `s3.ir-central1.arvanstorage.ir`. An anonymous request returns `AccessDenied` with `BucketName=filetransfer`; the request ID ends `ir-thr-at1`, so `ir-central1` is a newer hostname for the same Tehran cluster. Both hostnames resolve. | `curl` (no credentials), `dig` | The research notes' `ir-thr-at1` is stale. The SigV4 **signing region** string is **not verified** (needs a key pair). |
| Arvan's panel has a CORS form: Origin, Allowed Method (GET/PUT/POST/DELETE/HEAD), Allowed Header, Max Age. No "expose headers" field. | pasted panel text | We do not need `ETag` exposed: `putPart` only checks the status code and the API takes the part list from the bucket's ListParts at `complete`. |

## 2. Decisions

| # | Decision | Why |
|---|---|---|
| D1 | Stage on `ft.webcad.ir` first. `FT_HOST` and `PUBLIC_BASE_URL` in `app.env` are the only things to change when `filetransfer.ir` gets DNS. | Unblocks the deploy without waiting on IRNIC delegation. |
| D2 | Dedicated `ft-postgres` (18-alpine) and `ft-redis` containers, bind-mounted under `/opt/filetransfer`, never published to the host. | Isolation from Supabase's upgrades and outages. |
| D3 | Kavenegar for OTP SMS (already what the API supports). | Decided by the owner. |
| D4 | Arvan Object Storage bucket for files (virtual-host style, `OBJECT_STORAGE_PATH_STYLE=0`). | Real storage from day one; the pricing ticket does not block creating a bucket. |
| D5 | Traefik routes `/api/` to the Go API (priority 100) and everything else to the web container. | Next.js bakes its `rewrites()` at build time, so a runtime proxy to the API would be fragile; path routing keeps the session cookie same-origin. |
| D6 | Build on the Mac from `git archive <tag>`; refuse anything that is not an annotated tag on `origin/main`. | The box cannot build, and a tag is the only reviewed, immutable input. |

## 3. How a release works

1. `scripts/promote vX.Y.Z "note"` tags `origin/dev` and fast-forwards `main`.
2. `deploy/filetransfer/build.sh vX.Y.Z` builds both images for `linux/amd64`, checks the web `version.json` against the tag and commit, scans the web bundle for secret-looking strings, and writes `bundle.tar.gz` with a checksum and the layer digests.
3. `deploy/filetransfer/ship.sh vX.Y.Z` re-checks the checksum and that the tag still points at the built commit, uploads the bundle, verifies the upload checksum, then runs `ft-deploy` on the box.
4. `ft-deploy` (on the box) refuses unless: the tag and digests are well-formed, no other deploy is running, the tag was not used before, `app.env` is mode 600 with no placeholders left, base images exist, disk is under 80% and has room. Then it backs up Postgres, loads the images, **re-derives the layer digests and compares them to the ones built on the Mac**, starts the stack, waits for all four containers to be healthy, and proves the release through Traefik with real TLS:
   - `/version.json` shows this commit and version
   - `/login` is HTML and the `X-Robots-Tag` header matches `FT_ROBOTS`
   - `/api/v1/auth/me` returns the API's own JSON 401 (proves `/api/` reaches the Go API)
   - the `users` and `files` tables exist (migrations ran)
   - nothing listens on host ports 5432 or 6379
5. Any failed check restores the previous release and waits for it to be healthy. With no previous release it stops the stack and leaves the data untouched.

## 4. Prerequisites only a human can do (the deploy is gated on these)

State as of 2026-10-23 03:40 +0330:

- [x] Merge #7, #11, #13, #14, #15 into `dev` (#10 was closed by GitHub, replaced by #14).
- [ ] Cut the first tag: `scripts/promote v0.1.0`. The main checkout must be clean (it has one uncommitted `.gitignore` line, `.claude-drafts/`: discard or PR it first).
- [x] DNS: `ft.webcad.ir` -> `85.198.52.103` (explicit record, proxy off).
- [x] Kavenegar API key and template copied on the box from the webcad Supabase env (`/opt/webcad/supabase/.env`, `KAVENEGAR_TEMPLATE` -> `KAVENEGAR_VERIFY_TEMPLATE`). **The template is `houshkar`, another product's: users would see its wording.** Replace it with a filetransfer-specific, Kavenegar-approved template before launch (approval takes time; one line in `app.env`).
- [x] Arvan bucket `filetransfer` created (settings: Standard, Tehran, multi-zone off, Object Lock off, private, versioning off).
- [ ] **Arvan access key pair** (`OBJECT_STORAGE_ACCESS_KEY`, `OBJECT_STORAGE_SECRET_KEY`): still `CHANGE_ME`, which makes `ft-deploy` refuse to run. This is the only placeholder left.
- [ ] CORS rule in the Arvan panel: origin `https://ft.webcad.ir`, methods `PUT`/`GET`/`HEAD`, allowed header `*`, max age `3600`. Then check from outside with an unauthenticated `OPTIONS` preflight (see the handoff). Without CORS every browser upload part is blocked, which we reproduced locally.
- [ ] Verify the signing region with the new keys (`aws s3api head-bucket --endpoint-url https://s3.ir-central1.arvanstorage.ir --region ir-central1`, then `ir-thr-at1`). `app.env` currently has `ir-central1` for endpoint, public endpoint and region.
- [x] `deploy/filetransfer/bootstrap.sh` ran on the box: `/opt/filetransfer` tree, `app.env` (mode 600, generated Postgres password and `OTP_SECRET`), base images pulled, `/etc/cron.d/filetransfer-backup` at 03:45. Existing containers were untouched.

## 5. Not covered / known risks

- Not exercised against the real box yet: `build.sh`, `ship.sh` and `ft-deploy` have only had syntax, lint, local image builds, the backup script (success and failure), the env generator and the placeholder check tested. `bootstrap.sh` has run for real.
- Found in the bootstrap dry run, before it touched the box: `pgdata` was created `0700 root:root`, but it is bind-mounted at `/var/lib/postgresql` and Postgres runs as `postgres`, so first start failed with `mkdir: can't create directory '/var/lib/postgresql/18/': Permission denied`. Fixed in #15 (`0755`; Postgres locks its own versioned directory to `0700`). Reproduced and verified with a root-owned tmpfs, because a Docker Desktop bind mount does not reproduce the ownership. `redisdata` (`0700`) works: the Redis entrypoint chowns `/data`.
- The signing region (`ir-central1` vs `ir-thr-at1`) is unverified. A wrong value passes the deploy health check and then fails every browser upload with a signature error, so test it before shipping.
- `GET /api/v1/config` returns `"uploadProtocol":"tus"`, a stale string from the scaffold; uploads are S3 multipart. Harmless, but fix it before anything reads it.
- Abandoned multipart uploads leave parts in the bucket (billed) until aborted. Not checked: whether the API reaps them, and whether Arvan supports an abort-incomplete-multipart lifecycle rule.
- The box is shared with webcad.ir; a bad deploy here can only affect the `ft-*` containers, but a full disk or memory exhaustion would affect both. Memory limits and the 80% disk gate are the guards.
- `sshd` on the box has contradictory `PasswordAuthentication` lines across its config files. Out of scope here; worth a separate look.
- Redis persistence is on (AOF) so sessions survive restarts; losing it only signs users out.
- `filetransfer.ir` itself still needs nameservers at IRNIC before launch (D1).
