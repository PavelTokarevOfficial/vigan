# Media pipeline

Артефакты лежат в object storage: `sources/{clipID}/source.mp4`, `audio/{clipID}/audio.wav`, `subtitles/{clipID}/subtitles.srt` и `renders/{clipID}/vertical.mp4`.

Worker берёт job транзакционно, скачивает исходный ролик через Chromium/Rod, извлекает mono WAV 16 kHz через FFmpeg, передаёт WAV в `whisper-cli` и получает SRT. Далее FFmpeg собирает vertical layout: размытый фон, исходный ролик по центру и burned-in SRT. Для этого образ намеренно проверяет FFmpeg-фильтр `subtitles` (он требует сборку с libass); FFmpeg без него не подходит для worker. Временные файлы существуют только в каталоге job и удаляются после него.

Каждый устойчивый результат проверяется по ключу до работы: source/audio/subtitle/render можно безопасно переиспользовать при retry. Перед запуском каждого отсутствующего шага worker обновляет `processing_jobs.current_step` и процент прогресса (`extracting_audio`, `transcribing`, `rendering`). Пути FFmpeg и Whisper задаются environment variables, а не зашиты в код. Размер canvas, blur и preset кодека настраиваются через `OUTPUT_WIDTH`, `OUTPUT_HEIGHT`, `BACKGROUND_BLUR` и `FFMPEG_PRESET`.

Это поведение зафиксировано unit test-ом `internal/processing/runner_test.go`: retry не должен повторно скачивать исходник, извлекать audio, вызывать Whisper или перерендеривать готовый вариант.
