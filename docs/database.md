# Database

Schema is versioned by `golang-migrate` files in `back/migrations`. `streamers.twitch_login` and `clips.twitch_clip_id` are unique. `streamers.priority` — неотрицательное целое число, по которому стримеры сортируются от более приоритетных к менее приоритетным. Binary media is never stored in PostgreSQL: `media_files.storage_key` points to object storage. `processing_jobs` is the durable queue.

## Processing jobs

Для одного clip не может существовать два активных job одного типа (`pending` или `running`): это обеспечено partial unique index `processing_jobs_active_unique`. Ограничение дополняет `FOR UPDATE SKIP LOCKED` и защищает от двух одновременных API-запросов.

Состояния `clips.status` уже представляют доску без дополнительных таблиц: `saved` — избранное, `downloading` — скачивание в очереди/работе, `downloaded` — исходник есть в object storage, `completed` — есть render. При удалении избранного, скачанного, failed или completed clip внешний объект удаляется по каждому `media_files.storage_key`, после чего удаление строки `clips` каскадно очищает `media_files` и `processing_jobs`.

## Assets and templates

`asset_folders` и `assets` описывают виртуальное дерево. У каждого файла есть неизменяемый `storage_key` вида `assets/{assetID}/original`: rename и move изменяют только PostgreSQL, а не копируют объект в S3. `video_templates.config` хранит валидируемую версию JSON-композиции, а `template_asset_references` не даёт удалить ассет, который ещё используется шаблоном.

При `POST /process` в `processing_jobs` записываются `template_id` и `template_snapshot`. Snapshot содержит конфигурацию и конкретные S3-ключи ассетов, поэтому queued/running job не зависит от последующего rename или редактирования шаблона. Render metadata привязана к `processing_job_id`; ключ результата включает ID job, поэтому несколько рендеров одного clip не перезаписывают друг друга.
