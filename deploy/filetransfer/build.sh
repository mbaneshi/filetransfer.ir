#!/usr/bin/env bash
# Build the API and web images for a release tag on this Mac and verify them before anything ships.
# Usage: deploy/filetransfer/build.sh vX.Y.Z
# Output: $FT_OUT/<tag>/bundle.tar.gz plus api.id, web.id, commit. Never prints a secret.
# The box (3.9GB RAM) cannot build images, so they are built here and shipped as a tarball.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

TAG="${1:?usage: build.sh vX.Y.Z}"
OUT="${FT_OUT:-$HOME/.filetransfer-deploy}/$TAG"
die() { echo "build refused: $*" >&2; exit 1; }

[[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || die "tag $TAG does not match vX.Y.Z"
git fetch -q origin --tags
[ "$(git cat-file -t "refs/tags/$TAG" 2>/dev/null)" = tag ] || die "$TAG is not an annotated tag (scripts/promote makes one)"
COMMIT=$(git rev-parse "$TAG^{commit}")
git merge-base --is-ancestor "$COMMIT" origin/main || die "$TAG is not on origin/main (scripts/promote puts it there)"
for f in apps/api/Dockerfile apps/web/Dockerfile apps/web/pnpm-lock.yaml; do
  git cat-file -e "$TAG:$f" 2>/dev/null || die "$TAG has no $f"
done

# Build from the tag's exact tree, never the working directory.
SRC=$(mktemp -d)
CHECK=$(mktemp -d)
CID=""
cleanup() { [ -n "$CID" ] && docker rm "$CID" >/dev/null 2>&1 || true; rm -rf "$SRC" "$CHECK"; }
trap cleanup EXIT
git archive "$TAG" | tar -x -C "$SRC"

echo "== building $TAG ($COMMIT) for linux/amd64"
docker build --platform linux/amd64 -t "filetransfer-api:$TAG" "$SRC/apps/api"
docker build --platform linux/amd64 \
  --build-arg FT_VERSION="${TAG#v}" --build-arg FT_COMMIT="$COMMIT" \
  -t "filetransfer-web:$TAG" "$SRC/apps/web"

echo "== verifying the built images"
CID=$(docker create --platform linux/amd64 "filetransfer-web:$TAG")
docker cp "$CID:/app/public/version.json" "$CHECK/version.json"
for want in "\"commit\":\"$COMMIT\"" "\"version\":\"${TAG#v}\""; do
  grep -Fq "$want" "$CHECK/version.json" || die "web version.json does not contain $want"
done
docker cp "$CID:/app/.next" "$CHECK/next" >/dev/null
if grep -rIlE 'minioadmin|OTP_SECRET|KAVENEGAR|CHANGE_ME' "$CHECK/next" >/dev/null; then
  die "a secret-looking string is in the web bundle"
fi
docker rm "$CID" >/dev/null; CID=$(docker create --platform linux/amd64 "filetransfer-api:$TAG")
docker cp "$CID:/filetransfer" "$CHECK/filetransfer"
[ -s "$CHECK/filetransfer" ] || die "the API image has no server binary"

mkdir -p "$OUT"
layers() { docker image inspect "$1" --format '{{json .RootFS.Layers}}' | shasum -a 256 | cut -d' ' -f1; }
layers "filetransfer-api:$TAG" > "$OUT/api.id"
layers "filetransfer-web:$TAG" > "$OUT/web.id"
echo "$COMMIT" > "$OUT/commit"
docker save "filetransfer-api:$TAG" "filetransfer-web:$TAG" | gzip > "$OUT/bundle.tar.gz"
(cd "$OUT" && shasum -a 256 bundle.tar.gz > bundle.tar.gz.sha256)
echo "== built: $OUT ($(du -h "$OUT/bundle.tar.gz" | cut -f1)), api $(cut -c1-12 "$OUT/api.id") web $(cut -c1-12 "$OUT/web.id")"
