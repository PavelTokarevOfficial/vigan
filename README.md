# Finde Clip v2

Локальный modular monolith для импорта Twitch Clips и сборки вертикальных роликов с субтитрами.

## Prerequisites

- Docker Compose;
- Go 1.24+;
- Node.js 20+;
- FFmpeg, Chromium и `whisper-cli` для worker;
- локальная Whisper модель вне Git, например `models/ggml-tiny.bin`.

## Local setup

```bash
cp .env.example .env
docker compose up -d postgres minio
cd back && go run ./cmd/api
cd back && go run ./cmd/worker
cd front && npm install && npm run dev
```

API применяет migrations при запуске. Frontend Vite доступен локально; он проксирует `/api` на `localhost:8080`. MinIO console: `http://localhost:9001`.

## Checks

```bash
cd back && go test ./...
cd front && npm run build
```

Подробности: [architecture](docs/architecture.md), [backend](docs/backend.md), [frontend](docs/frontend.md), [database](docs/database.md), [media pipeline](docs/media-pipeline.md), [Whisper](docs/whisper.md). Текущий прогресс: [TASK_STATUS.md](TASK_STATUS.md).
