# Architecture

API and worker are separate Go processes sharing one module. PostgreSQL holds streamers, clips, media metadata and jobs. The `media.Storage` boundary has an S3-compatible adapter, so MinIO, AWS S3 and R2 can be used without changing application logic. The worker claims pending jobs transactionally with `FOR UPDATE SKIP LOCKED` and executes Download → ExtractAudio → Transcribe → Render as restartable steps.
