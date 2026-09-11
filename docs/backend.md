# Backend

`cmd/api` обслуживает REST API, применяет migrations и не запускает тяжёлые операции. `cmd/worker` claim-ит PostgreSQL jobs с `FOR UPDATE SKIP LOCKED` и запускает browser/media adapters.

Основные маршруты: CRUD `/api/streamers`, пакетное добавление ников `POST /api/streamers/bulk`, установка неотрицательного приоритета `PATCH /api/streamers/{id}/priority` с телом `{ "priority": 3 }`, удалённые Twitch clips `/api/streamers/{id}/clips?startedAt=YYYY-MM-DD&endedAt=YYYY-MM-DD` (если даты не переданы — последние 7 дней), импорт в избранное `/api/clips/import`, `POST /api/clips/{id}/download`, `POST /api/clips/{id}/process`, `POST /api/clips/{id}/retry`, `DELETE /api/clips/{id}`, наблюдение за очередью `/api/jobs` и `/api/jobs/{id}`, список готовых рендеров `/api/videos`. Для шаблонов есть CRUD и `POST /api/templates/{id}/duplicate`; для ассетов — list/upload/rename/move/delete `/api/assets` и CRUD `/api/asset-folders`. Команды enqueue не выполняют тяжёлую работу в HTTP handler: worker забирает созданную job из PostgreSQL.

External boundaries: `infrastructure/twitch`, `infrastructure/browser`, `infrastructure/storage`, `infrastructure/ffmpeg`, `infrastructure/whisper`. Application code использует ports в `internal/processing` и `internal/media`.

S3 adapter не трактует произвольную ошибку как отсутствие файла: `Exists` распознаёт только типизированный S3 `NotFound`/`NoSuchKey` или HTTP 404. Это сохраняет идемпотентность pipeline и не скрывает проблемы сети или credentials.

Конфигурация только через environment variables из `.env.example`. При локальном `cd back && go run ./cmd/api` или `go run ./cmd/worker` приложение автоматически читает `../.env` и заменяет Compose-hostnames `postgres`/`minio` на published `localhost` ports. В Docker имена сервисов остаются без изменений; Docker Compose передаёт свои переменные напрямую, и они имеют приоритет. API не возвращает stack traces или secrets. Повторный импорт существующего Twitch clip оставляет его в избранном и не создаёт download job. Скачивание создаётся только отдельной командой; одновременно для одного clip допускается только один активный job каждого типа. При смене Twitch login cached Twitch user ID очищается и будет заново разрешён перед следующим запросом clips.

Удаление избранного, скачанного, упавшего или готового клипа проходит через `media.Library`: сначала удаляются все ключи этого clip из S3-compatible storage, затем каскадно удаляются `clips`, `media_files` и `processing_jobs` в PostgreSQL. Нельзя удалить клип, который сейчас скачивается или обрабатывается.

S3 uses two endpoints: `S3_ENDPOINT` is the internal service address used by API/worker, while `S3_PUBLIC_ENDPOINT` is used only to sign browser URLs. In local Docker this is `http://localhost:9000`, not the internal `minio:9000` hostname.

Worker получает `SIGINT`/`SIGTERM` через context. Если контекст отменён во время job, job переводится обратно в `pending` с шагом `interrupted`, а не помечается как failed; последующий worker продолжит pipeline с уже сохранённых artifacts.

Worker пишет JSON structured logs для начала, каждого шага (`download`, `extracting_audio`, `transcribing`, `rendering`), завершения, ошибки и длительности job. В полях лога есть `job_id`, `clip_id`, `step` и `progress`.

`POST /api/clips/{id}/process` требует JSON `{ "templateId": "UUID" }`. API валидирует template config и сохраняет snapshot вместе с job. Worker скачивает только S3 keys из snapshot во временную папку, строит filter graph из декларативных слоёв и передаёт его в FFmpeg adapter. В HTTP response никогда не попадают credentials или внутренний S3 endpoint.
