#!/usr/bin/env bash
# One-time (and safe to re-run) setup of the box for filetransfer.ir. Run from the repo root on the Mac:
#   deploy/filetransfer/bootstrap.sh            # does it
#   FT_DRY_RUN=1 deploy/filetransfer/bootstrap.sh   # prints the remote script and stops
# It creates /opt/filetransfer, generates the database and OTP secrets into /opt/filetransfer/app.env (mode 600,
# never printed, never overwritten once it exists), pulls the Postgres and Redis images, and installs the
# nightly backup. It does not touch Traefik, Supabase, webcad.ir or any existing backup job.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

BOX="${FT_SSH:-root@85.198.52.103}"
KEY="${FT_KEY:-$HOME/.ssh/id_ed25519_arvan_webcad}"
SSH=(ssh -i "$KEY" -o IdentitiesOnly=yes)
SCP=(scp -q -i "$KEY" -o IdentitiesOnly=yes)

read -r -d '' REMOTE <<'EOS' || true
set -euo pipefail
ROOT=/opt/filetransfer
install -d -m 755 "$ROOT" "$ROOT/bin" "$ROOT/app" "$ROOT/app/incoming"
install -d -m 700 "$ROOT/backups" "$ROOT/pgdata" "$ROOT/redisdata"

if [ -f "$ROOT/app.env" ]; then
  echo "app.env exists: left untouched"
else
  pw=$(openssl rand -hex 24); otp=$(openssl rand -hex 32)
  sed -e "s/^POSTGRES_PASSWORD=CHANGE_ME/POSTGRES_PASSWORD=$pw/" \
      -e "s#filetransfer:CHANGE_ME@#filetransfer:$pw@#" \
      -e "s/^OTP_SECRET=CHANGE_ME/OTP_SECRET=$otp/" \
      "$ROOT/app.env.example" > "$ROOT/app.env.new"
  chmod 600 "$ROOT/app.env.new"; mv "$ROOT/app.env.new" "$ROOT/app.env"
  echo "app.env created with generated secrets; still to fill in by hand: $(grep -E "^[A-Z_]+=.*CHANGE_ME" "$ROOT/app.env" | cut -d= -f1 | tr '\n' ' ')"
fi

docker pull postgres:18-alpine >/dev/null && echo "pulled postgres:18-alpine"
docker pull redis:8-alpine >/dev/null && echo "pulled redis:8-alpine"

install -m 755 "$ROOT/backup.sh.new" "$ROOT/backup.sh" && rm -f "$ROOT/backup.sh.new"
echo '45 3 * * * root /opt/filetransfer/backup.sh >> /var/log/ft-backup.log 2>&1' > /etc/cron.d/filetransfer-backup
chmod 644 /etc/cron.d/filetransfer-backup
echo "bootstrap done"
EOS

if [ "${FT_DRY_RUN:-}" = 1 ]; then
  echo "[dry-run] would copy app.env.example and backup.sh to $BOX:/opt/filetransfer, then run on the box:"
  echo "$REMOTE"
  exit 0
fi

"${SSH[@]}" "$BOX" 'mkdir -p /opt/filetransfer'
"${SCP[@]}" deploy/filetransfer/app.env.example "$BOX:/opt/filetransfer/app.env.example"
"${SCP[@]}" deploy/filetransfer/backup.sh "$BOX:/opt/filetransfer/backup.sh.new"
"${SSH[@]}" "$BOX" 'bash -s' <<<"$REMOTE"
