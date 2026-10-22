# Deploy filetransfer.ir to the Arvan box (RFC record)

Status: decided 2026-10-23, scripts written, **first real deploy not yet run**. Issue: #12.
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
| `scripts/promote` exists; repo has 0 tags; `main` is the scaffold | `git tag`, `scripts/promote` | Nothing is releasable until #7, #10, #11 merge and a tag is promoted. |

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

- [ ] Merge #7, #10, #11 (in that order), then cut the first tag with `scripts/promote`.
- [ ] DNS: an `A` record `ft.webcad.ir` -> `85.198.52.103` (needed before Let's Encrypt can issue the certificate).
- [ ] Kavenegar: an account, API key, and an approved verify/lookup template that takes one token.
- [ ] Arvan: a bucket, an access key pair, and a CORS rule for the app origin (`PUT`, `GET`, `HEAD`; `AllowedHeader *`; expose `ETag`). Without CORS every browser upload part is blocked, which we reproduced locally.
- [ ] Run `deploy/filetransfer/bootstrap.sh`, then edit the five placeholders it lists in `/opt/filetransfer/app.env` on the box.

## 5. Not covered / known risks

- Not exercised against the real box yet: only syntax, lint, local image builds, the backup script (success and failure), the env generator and the placeholder check have been tested.
- The box is shared with webcad.ir; a bad deploy here can only affect the `ft-*` containers, but a full disk or memory exhaustion would affect both. Memory limits and the 80% disk gate are the guards.
- `sshd` on the box has contradictory `PasswordAuthentication` lines across its config files. Out of scope here; worth a separate look.
- Redis persistence is on (AOF) so sessions survive restarts; losing it only signs users out.
- `filetransfer.ir` itself still needs nameservers at IRNIC before launch (D1).
