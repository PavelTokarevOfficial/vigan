# Frontend

Vue 3 + TypeScript + Vite + Pinia + Vue Router + Tailwind. Код размещён в FSD-каталогах: `app`, `pages`, `widgets`, `features`, `entities`, `shared`.

Сейчас реализованы layout, navigation, CRUD Streamers, выбор и сохранение Twitch Clips (включая отметку уже импортированных), Pipeline с выбором баннера и live status, список готовых видео с оригиналом и download, а также upload/list/delete баннеров. FSD используется фактически: `widgets/AppHeader`, `features/clip-import/ClipImportButton`, DTO-типы в `entities`, а общие `AppButton`, `EmptyState`, `ErrorState` и HTTP helper — в `shared`. UI libraries не используются. Production проверка: `npm run build`.
