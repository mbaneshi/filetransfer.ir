# Handoff 2026-10-23: deploy is ready except the Arvan access-key pair

Read this first, then `docs/superpowers/specs/2026-10-23-deploy-arvan-box.md` (decisions D1-D6, release flow, risks).

## State in one table

| Piece | State |
|---|---|
| Code | All merged on `dev` (`9c897a6` at handoff, plus this docs PR). No open PRs. No tags. `origin/main` does not exist yet. |
| Box `root@85.198.52.103` (shared with webcad.ir) | Bootstrapped. `/opt/filetransfer` tree, `app.env` (mode 600), base images, backup cron at 03:45. **0 filetransfer containers running.** Traefik, Supabase, webcad untouched. |
| DNS | `ft.webcad.ir` -> `85.198.52.103` (explicit record; the zone has a wildcard to another IP that this overrides). Arvan proxy OFF. |
| `app.env` placeholders left | `OBJECT_STORAGE_ACCESS_KEY`, `OBJECT_STORAGE_SECRET_KEY`. Both are `CHANGE_ME`, so `ft-deploy` refuses to run (the safe state). |
| Kavenegar | Key and template copied from the webcad Supabase env. Template `houshkar` belongs to another product: replace before launch. |
| Bucket `filetransfer` | Exists, private, Standard, Tehran. Endpoint/region in `app.env`: `s3.ir-central1.arvanstorage.ir` / `ir-central1`. |
| CORS | **Not set.** |
| Signing region | **Unverified** (needs a key pair). |
| Live site | `https://ft.webcad.ir` -> Traefik 404 (nothing routed). |

## Do these, in order

1. **Make the access-key pair.** In the Arvan panel: Object Storage -> access keys (NOT the general "API keys" page). Scope to bucket `filetransfer` if offered, read+write. The secret is usually shown once.
   Past attempts pasted ordinary text (lengths 144, 165, 138, 229, 439; spaces and `=` signs) instead of a key, probably from the wrong page or from terminal text. A key is one dense token with no spaces.
2. **Send each value to the box without printing it.** Re-create the helper on the box (it was deleted), then pipe a hidden-input dialog into it. The Bash tool cannot do this itself (see "Permissions"): the user runs these with the `!` prefix.

   Helper `/opt/filetransfer/ft-set-key.sh` (copy it with `scp`):

   ```bash
   set -euo pipefail
   name="${1:-}"
   case "$name" in OBJECT_STORAGE_ACCESS_KEY|OBJECT_STORAGE_SECRET_KEY) ;; *) echo "bad name"; exit 1 ;; esac
   IFS= read -r V || true
   V=$(printf '%s' "$V" | tr -d '[:space:]')
   [ -n "$V" ] || { echo "empty value: nothing written"; exit 1; }
   dst=/opt/filetransfer/app.env
   export V name
   cp -p "$dst" "$dst.bak3"
   awk 'BEGIN { FS = OFS = "=" } $1 == ENVIRON["name"] { print $1, ENVIRON["V"]; next } { print }' "$dst.bak3" > "$dst.new"
   chmod 600 "$dst.new"; mv "$dst.new" "$dst"
   echo "$name set: ${#V} characters (value not shown)"
   grep -E '^[A-Z_]+=.*CHANGE_ME' "$dst" | cut -d= -f1 || true
   ```

   Send a value (run once per key, with `OBJECT_STORAGE_SECRET_KEY` the second time):

   ```
   osascript -e 'text returned of (display dialog "Paste the ACCESS key" default answer "" with hidden answer)' | ssh -i ~/.ssh/id_ed25519_arvan_webcad -o IdentitiesOnly=yes root@85.198.52.103 'bash /opt/filetransfer/ft-set-key.sh OBJECT_STORAGE_ACCESS_KEY'
   ```

   Then delete the helper and the `app.env.bak3` it leaves. Do NOT use the awk-split length check on `app.env` values: it splits at `=` and under-reports.
3. **Set CORS in the Arvan panel** (bucket -> CORS): origin `https://ft.webcad.ir`, methods PUT, GET, HEAD, allowed header `*` (use `content-type` if `*` is refused), max age `3600`. No expose-header field exists and none is needed.
   Verify from outside, no credentials needed:
   ```
   curl -si -X OPTIONS https://filetransfer.s3.ir-central1.arvanstorage.ir/test -H 'Origin: https://ft.webcad.ir' -H 'Access-Control-Request-Method: PUT' | head
   ```
   Expect `Access-Control-Allow-Origin` and a PUT in `Access-Control-Allow-Methods`. Baseline before the rule exists (checked 2026-10-23): `HTTP/2 403`, no `Access-Control-*` headers.
4. **Settle the signing region** with the new keys, from the Mac (keys typed at the prompts, never into this chat):
   ```
   read -s "AK?access key: " && read -s "SK?secret key: " && export AWS_ACCESS_KEY_ID=$AK AWS_SECRET_ACCESS_KEY=$SK && for r in ir-central1 ir-thr-at1; do echo "== $r"; aws s3api head-bucket --bucket filetransfer --endpoint-url https://s3.ir-central1.arvanstorage.ir --region $r && echo OK; done
   ```
   If only `ir-thr-at1` works, change `OBJECT_STORAGE_REGION` (and the endpoint to match) in `app.env` on the box, and update `deploy/filetransfer/app.env.example`.
5. **Clean the main checkout, then tag.** `/Users/bm/filetransfer.ir` has one uncommitted line in `.gitignore` (`.claude-drafts/`); `scripts/promote` refuses a dirty tree. Discard it (`git checkout .gitignore`) or PR it. Then `scripts/promote v0.1.0 "first deploy"`.
6. **Deploy** from a worktree at the tag's commit (the main checkout has no `deploy/` until `main` advances): `deploy/filetransfer/build.sh v0.1.0`, then `ship.sh v0.1.0`. Ask the user before each. `ft-deploy` checks through Traefik with real TLS and rolls back on failure.
7. **Smoke test in a browser** at `https://ft.webcad.ir`: log in (a real SMS now), upload a file bigger than 64 MiB, share it, download it, and confirm the CORS/region path works end to end.

## Permissions: what the harness refuses, and the way around

- `gh` write commands (merge, ready, create, edit) and anything that runs `bootstrap.sh`/`ship.sh` on the box were denied by the auto-mode classifier. The user runs them with the `!` prefix. Read-only `gh`, read-only SSH checks, and plain `git` worktree/branch commands work.
- Reading credentials from other apps' files on the box was denied as "credential exploration". The workable pattern: discover names only, then have the user run a script that copies values on the box without printing them.
- `nano` over `ssh -t` fails from the `!` prompt (no terminal). Use the clipboard/dialog pipe above.
- Do not work around a denial. Report it and hand the user the exact command.

## Rules learned this session

- **Stacked PRs:** merging the lower PR with `--delete-branch` makes GitHub CLOSE the stacked one, not retarget it (it closed #10). Merge the lower PR without `--delete-branch`, retarget the upper PR (`gh pr edit N --base dev`), then delete the branch. After a squash merge, rebase the upper branch with `git rebase --onto origin/dev <old-base-sha>`, not a plain rebase (the plain one replays the already-squashed commits and conflicts).
- **Dry-run reading pays:** reading the printed remote script found the `pgdata` permission bug before it touched the box. Verify the claim in an environment that can reproduce it (a root-owned tmpfs), not on Docker Desktop's bind mounts.
- **Check a wildcard before trusting DNS:** `dig` a name that cannot exist as well as the real one.
- Local `git` through a worktree only: `bootstrap`/`build`/`ship` read `deploy/` from the working tree, so run them from a worktree at `origin/dev` (or the tag), never the main checkout while it sits on the scaffold commit.

## Follow-ups (not blocking the first deploy)

| Item | Detail |
|---|---|
| Own Kavenegar template | Replace `houshkar`; get one approved (takes time). |
| `uploadProtocol: "tus"` | Stale string in `GET /api/v1/config`; uploads are S3 multipart. |
| Abandoned multipart uploads | Check whether the API aborts stale ones; check Arvan lifecycle support. Parts are billed until aborted. |
| `filetransfer.ir` nameservers | Registered at IRNIC with none; delegate before launch, then change `FT_HOST` and `PUBLIC_BASE_URL`. |
| Wallet and usage ledger (#3, #4) | Blocked on Arvan ticket `24C-326D1343-0055` and the pricing test. |
| `sshd` on the box | Contradictory `PasswordAuthentication` lines across config files. Out of scope, flagged. |
| GitHub Discussions | Disabled on the repo; enabling is web-UI only. RFC-flow Discussions need it. |
| Rotate the Arvan key pair once | The access-key attempts may have put clipboard text into `app.env.bak*` (deleted). No real secret is known to have leaked, but a fresh pair costs nothing. |

## Where things are

- Persian digest of this session: `~/.claude/fa/2026-10-23-filetransfer-deploy-prep-access-key.html`.
- Deploy scripts: `deploy/filetransfer/` (`bootstrap.sh`, `build.sh`, `ship.sh`, `ft-deploy`, `backup.sh`, `compose.app.yml`, `app.env.example`).
- Session scratch files (`ft-copy-kavenegar.sh`, `ft-set-bucket.sh`, `ft-set-key.sh`) were temporary and are not in the repo; the key-setter is reproduced above.
