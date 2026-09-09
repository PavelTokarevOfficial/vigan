# Architecture

API and worker are separate Go processes sharing one module. PostgreSQL holds streamers, clips, media metadata, assets, templates and jobs. The `media.Storage` boundary has an S3-compatible adapter, so MinIO, AWS S3 and R2 can be used without changing application logic. The worker claims pending jobs transactionally with `FOR UPDATE SKIP LOCKED` and executes Download → ExtractAudio → Transcribe → Render as restartable steps.

Video templates are declarative, versioned JSON. The API validates a template before a process job is created and stores an immutable snapshot with the job; the worker therefore renders the version selected by the user, not a later edit of the template.
