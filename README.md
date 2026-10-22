# FileTransfer.ir

Iran-focused file sharing and hosting platform.

## MVP architecture

- `apps/web`: Next.js frontend (RTL Persian UI)
- `apps/api`: Go API service
- PostgreSQL: users, files, shares, plans, payments
- Redis: OTPs, sessions, rate limits, background jobs
- Object storage: Arvan Object Storage / S3-compatible storage
- `tusd`: resumable uploads for large files
- Caddy: HTTPS and reverse proxy

## Run locally

```bash
docker compose up --build
```

Local services use non-default host ports:

- Web: `http://localhost:18473`
- API health: `http://localhost:18474/health`
- MinIO API: `http://localhost:19000`
- MinIO console: `http://localhost:19001`
- PostgreSQL: `localhost:15432`
- Redis: `localhost:16379`

## Environment

Copy `.env.example` to `.env` and replace secrets before using anything outside local development.

