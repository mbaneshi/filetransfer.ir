#!/usr/bin/env bash
# Copy a built release to the box and run ft-deploy there.
# Usage: deploy/filetransfer/ship.sh vX.Y.Z   (after build.sh)
# FT_DRY_RUN=1 runs every local check and prints the remote steps instead of running them.
set -euo pipefail

TAG="${1:?usage: ship.sh vX.Y.Z}"
[[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || { echo "bad tag $TAG" >&2; exit 1; }
cd "$(git rev-parse --show-toplevel)"
git fetch -q origin --tags --force
BOX="${FT_SSH:-root@85.198.52.103}"
KEY="${FT_KEY:-$HOME/.ssh/id_ed25519_arvan_webcad}"
OUT="${FT_OUT:-$HOME/.filetransfer-deploy}/$TAG"
REMOTE=/opt/filetransfer
SSH=(ssh -i "$KEY" -o IdentitiesOnly=yes)
SCP=(scp -q -i "$KEY" -o IdentitiesOnly=yes)
run() { if [ "${FT_DRY_RUN:-}" = 1 ]; then echo "[dry-run] $*"; else "$@"; fi; }

[ "$(git rev-parse "$TAG^{commit}")" = "$(cat "$OUT/commit" 2>/dev/null)" ] || { echo "$TAG no longer points at the built commit; rebuild" >&2; exit 1; }
[ -f "$OUT/bundle.tar.gz" ] || { echo "no build for $TAG; run build.sh first" >&2; exit 1; }
(cd "$OUT" && shasum -a 256 -c bundle.tar.gz.sha256)

run "${SSH[@]}" "$BOX" "mkdir -p $REMOTE/bin $REMOTE/app/incoming"
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
git show "$TAG:deploy/filetransfer/ft-deploy" > "$STAGE/ft-deploy"
git show "$TAG:deploy/filetransfer/compose.app.yml" > "$STAGE/docker-compose.yml"
git show "$TAG:deploy/filetransfer/backup.sh" > "$STAGE/backup.sh"
run "${SCP[@]}" "$STAGE/ft-deploy" "$BOX:$REMOTE/bin/ft-deploy.new"
run "${SCP[@]}" "$STAGE/backup.sh" "$BOX:$REMOTE/backup.sh.new"
run "${SCP[@]}" "$STAGE/docker-compose.yml" "$BOX:$REMOTE/app/docker-compose.yml.new"
run "${SSH[@]}" "$BOX" "chmod 755 $REMOTE/bin/ft-deploy.new $REMOTE/backup.sh.new && mv $REMOTE/bin/ft-deploy.new $REMOTE/bin/ft-deploy && mv $REMOTE/backup.sh.new $REMOTE/backup.sh && mv $REMOTE/app/docker-compose.yml.new $REMOTE/app/docker-compose.yml"

run "${SCP[@]}" "$OUT/bundle.tar.gz" "$BOX:$REMOTE/app/incoming/$TAG.tar.gz"
if [ "${FT_DRY_RUN:-}" != 1 ]; then
  REMOTE_SUM=$("${SSH[@]}" "$BOX" "sha256sum $REMOTE/app/incoming/$TAG.tar.gz | cut -d' ' -f1")
  [ "$REMOTE_SUM" = "$(cut -d' ' -f1 "$OUT/bundle.tar.gz.sha256")" ] || { echo "upload checksum mismatch" >&2; exit 1; }
fi

run "${SSH[@]}" "$BOX" "$REMOTE/bin/ft-deploy $TAG $(cat "$OUT/api.id") $(cat "$OUT/web.id") $(cat "$OUT/commit") $REMOTE/app/incoming/$TAG.tar.gz"
