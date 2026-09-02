# TGMon UI

Next.js app for the TGMon media library: sign in, browse videos, and play them.

## Setup

```bash
npm install
```

Copy [`.env.example`](.env.example) to `.env.local`:

```
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_MINIO_URL=
```

- `NEXT_PUBLIC_API_URL` — Go API origin (JSON under `/api/`, streams at `/stream/:id`).
- `NEXT_PUBLIC_MINIO_URL` — public MinIO prefix used for thumbnails and sprite VTT

The Go server should allow this UI origin in CORS (`HTTP__CORES_ALLOWED_ORIGINS`) when not using allow-all.

## Develop

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).
