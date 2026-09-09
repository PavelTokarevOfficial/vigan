# Database

Schema is versioned by `golang-migrate` files in `back/migrations`. `streamers.twitch_login` and `clips.twitch_clip_id` are unique. Binary media is never stored in PostgreSQL: `media_files.storage_key` points to object storage. `processing_jobs` is the durable queue.

## Processing jobs

Для одного clip не может существовать два активных job одного типа (`pending` или `running`): это обеспечено partial unique index `processing_jobs_active_unique`. Ограничение дополняет `FOR UPDATE SKIP LOCKED` и защищает от двух одновременных API-запросов.

Состояния `clips.status` уже представляют доску без дополнительных таблиц: `saved` — избранное, `downloading` — скачивание в очереди/работе, `downloaded` — исходник есть в object storage, `completed` — есть render. При удалении избранного, скачанного, failed или completed clip внешний объект удаляется по каждому `media_files.storage_key`, после чего удаление строки `clips` каскадно очищает `media_files` и `processing_jobs`.
