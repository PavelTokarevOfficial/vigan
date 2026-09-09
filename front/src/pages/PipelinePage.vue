<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import AppButton from '../shared/ui/AppButton.vue'
import ErrorState from '../shared/ui/ErrorState.vue'
import { readError } from '../shared/api/http'
import type { VideoTemplate } from '../entities/template/model/types'

type Clip = {
  id: string
  title: string
  streamerName: string
  thumbnailURL: string
  status: string
  error: string
  currentStep: string
  progress: number
  lastJobType: string
  lastJobStatus: string
}

type Video = {
  clipId: string
  title: string
  streamer: string
  twitchUrl: string
  url: string
  createdAt: string
}

type Column = 'downloaded' | 'ready'

const clips = ref<Clip[]>([])
const videos = ref<Video[]>([])
const error = ref('')
const busy = ref('')
const dragged = ref<Clip | null>(null)
const queuedProcessIDs = ref(new Set<string>())
const processClipID = ref<string | null>(null)
const templates = ref<VideoTemplate[]>([])
const templatesLoading = ref(false)

const videosByClip = computed(
  () => new Map(videos.value.map((video) => [video.clipId, video])),
)
const favorites = computed(() =>
  clips.value.filter(
    (clip) =>
      clip.status === 'saved' ||
      clip.status === 'downloading' ||
      (clip.status === 'failed' && clip.lastJobType === 'download'),
  ),
)
const downloaded = computed(() =>
  clips.value.filter(
    (clip) =>
      ['downloaded', 'transcribing', 'ready_to_render', 'rendering'].includes(
        clip.status,
      ) ||
      (clip.status === 'failed' && clip.lastJobType === 'process'),
  ),
)
const completed = computed(() =>
  clips.value.filter((clip) => clip.status === 'completed'),
)

async function load() {
  try {
    const [clipResponse, videoResponse] = await Promise.all([
      fetch('/api/clips'),
      fetch('/api/videos'),
    ])
    if (!clipResponse.ok || !videoResponse.ok) {
      throw new Error('Не удалось загрузить доску')
    }
    clips.value = (await clipResponse.json()).data || []
    videos.value = (await videoResponse.json()).data || []
    const downloadedIDs = new Set(
      clips.value
        .filter((clip) => clip.status === 'downloaded')
        .map((clip) => clip.id),
    )
    queuedProcessIDs.value = new Set(
      [...queuedProcessIDs.value].filter((id) => downloadedIDs.has(id)),
    )
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось загрузить доску'
  }
}

async function action(
  id: string,
  path: 'download' | 'process' | 'retry',
  templateID?: string,
) {
  busy.value = id
  error.value = ''
  const response = await fetch(`/api/clips/${id}/${path}`, {
    method: 'POST',
    ...(path === 'process'
      ? {
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ templateId: templateID }),
        }
      : {}),
  })
  if (!response.ok) {
    error.value = await readError(
      response,
      'Не удалось поставить задачу в очередь',
    )
  } else if (path === 'process') {
    queuedProcessIDs.value = new Set([...queuedProcessIDs.value, id])
  }
  busy.value = ''
  await load()
}

async function openTemplateChooser(id: string) {
  error.value = ''
  templatesLoading.value = true
  processClipID.value = id
  try {
    const response = await fetch('/api/templates')
    if (!response.ok)
      throw new Error(await readError(response, 'Не удалось загрузить шаблоны'))
    templates.value = (await response.json()).data || []
    if (!templates.value.length) {
      error.value = 'Сначала создайте хотя бы один шаблон видео.'
      processClipID.value = null
    }
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось загрузить шаблоны'
    processClipID.value = null
  } finally {
    templatesLoading.value = false
  }
}

async function chooseTemplate(templateID: string) {
  const id = processClipID.value
  if (!id) return
  await action(id, 'process', templateID)
  processClipID.value = null
}

async function remove(clip: Clip) {
  busy.value = clip.id
  error.value = ''
  const response = await fetch(`/api/clips/${clip.id}`, { method: 'DELETE' })
  if (!response.ok) {
    error.value = await readError(response, 'Не удалось удалить клип')
  }
  busy.value = ''
  await load()
}

function canDelete(clip: Clip) {
  return (
    ['saved', 'downloaded', 'failed'].includes(clip.status) &&
    !isProcessQueued(clip)
  )
}

function isProcessQueued(clip: Clip) {
  return (
    clip.status === 'downloaded' &&
    (queuedProcessIDs.value.has(clip.id) ||
      (clip.lastJobType === 'process' &&
        ['pending', 'running'].includes(clip.lastJobStatus)))
  )
}

function drop(column: Column) {
  const clip = dragged.value
  dragged.value = null
  if (!clip) return

  if (column === 'downloaded' && clip.status === 'saved') {
    void action(clip.id, 'download')
  }
  if (
    column === 'ready' &&
    clip.status === 'downloaded' &&
    !isProcessQueued(clip)
  ) {
    void openTemplateChooser(clip.id)
  }
}

function statusText(clip: Clip) {
  if (clip.status === 'saved') return 'В избранном'
  if (isProcessQueued(clip)) {
    return clip.lastJobStatus === 'running'
      ? `${clip.currentStep || 'Запуск обработки'} · ${clip.progress}%`
      : 'В очереди на обработку'
  }
  if (clip.status === 'downloaded') return 'Готов к обработке'
  if (clip.status === 'completed') return 'Готово'
  if (clip.status === 'failed') return 'Ошибка'
  return `${clip.currentStep || clip.status} · ${clip.progress}%`
}

let timer: number | undefined
onMounted(() => {
  void load()
  timer = window.setInterval(load, 3000)
})
onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <section>
    <h2 class="text-xl font-semibold">Pipeline</h2>
    <p class="mt-1 text-slate-600">
      Перетащите избранный клип в «Скаченное», чтобы скачать его, а скаченный —
      в «Готовые видео», чтобы запустить обработку.
    </p>

    <ErrorState v-if="error" class="mt-4" :message="error" />

    <div class="mt-6 grid gap-4 xl:grid-cols-4">
      <section class="rounded-lg border border-slate-200 bg-slate-50 p-3">
        <h3 class="font-semibold">Избранное · {{ favorites.length }}</h3>
        <p class="mt-1 text-sm text-slate-500">Можно удалить или скачать.</p>
        <div class="mt-3 space-y-3">
          <p v-if="!favorites.length" class="text-sm text-slate-500">
            Пока пусто.
          </p>
          <article
            v-for="clip in favorites"
            :key="clip.id"
            draggable="true"
            class="cursor-grab rounded bg-white p-3 shadow-sm active:cursor-grabbing"
            @dragstart="dragged = clip"
            @dragend="dragged = null"
          >
            <img
              v-if="clip.thumbnailURL"
              :src="clip.thumbnailURL"
              :alt="`Превью клипа: ${clip.title}`"
              class="mb-3 aspect-video w-full rounded object-cover"
            >
            <b>{{ clip.title }}</b>
            <p class="mt-1 text-sm text-slate-600">
              {{ clip.streamerName }} · {{ statusText(clip) }}
            </p>
            <p v-if="clip.error" class="mt-1 text-sm text-red-600">
              {{ clip.error }}
            </p>
            <div class="mt-3 flex flex-wrap gap-2">
              <AppButton
                v-if="clip.status === 'saved'"
                :disabled="busy === clip.id"
                @click="action(clip.id, 'download')"
              >
                Скачать
              </AppButton>
              <AppButton
                v-if="clip.status === 'failed'"
                :disabled="busy === clip.id"
                @click="action(clip.id, 'retry')"
              >
                Повторить
              </AppButton>
              <AppButton
                v-if="canDelete(clip)"
                variant="danger"
                :disabled="busy === clip.id"
                @click="remove(clip)"
              >
                Удалить
              </AppButton>
            </div>
          </article>
        </div>
      </section>

      <!-- biome-ignore lint/a11y/noStaticElementInteractions: native drop target for the drag-and-drop board -->
      <section
        class="rounded-lg border border-slate-200 bg-slate-50 p-3"
        @dragover.prevent
        @drop="drop('downloaded')"
      >
        <h3 class="font-semibold">Скаченное · {{ downloaded.length }}</h3>
        <p class="mt-1 text-sm text-slate-500">
          Можно удалить или пустить в работу.
        </p>
        <div class="mt-3 space-y-3">
          <p v-if="!downloaded.length" class="text-sm text-slate-500">
            Пока пусто.
          </p>
          <article
            v-for="clip in downloaded"
            :key="clip.id"
            draggable="true"
            class="cursor-grab rounded bg-white p-3 shadow-sm active:cursor-grabbing"
            @dragstart="dragged = clip"
            @dragend="dragged = null"
          >
            <img
              v-if="clip.thumbnailURL"
              :src="clip.thumbnailURL"
              :alt="`Превью клипа: ${clip.title}`"
              class="mb-3 aspect-video w-full rounded object-cover"
            >
            <b>{{ clip.title }}</b>
            <p class="mt-1 text-sm text-slate-600">
              {{ clip.streamerName }} · {{ statusText(clip) }}
            </p>
            <p v-if="clip.error" class="mt-1 text-sm text-red-600">
              {{ clip.error }}
            </p>
            <div class="mt-3 flex flex-wrap gap-2">
              <AppButton
                v-if="clip.status === 'downloaded' && !isProcessQueued(clip)"
                :disabled="busy === clip.id"
                @click="openTemplateChooser(clip.id)"
              >
                В работу
              </AppButton>
              <AppButton
                v-if="clip.status === 'failed'"
                :disabled="busy === clip.id"
                @click="action(clip.id, 'retry')"
              >
                Повторить
              </AppButton>
              <AppButton
                v-if="canDelete(clip)"
                variant="danger"
                :disabled="busy === clip.id"
                @click="remove(clip)"
              >
                Удалить
              </AppButton>
            </div>
          </article>
        </div>
      </section>

      <!-- biome-ignore lint/a11y/noStaticElementInteractions: native drop target for the drag-and-drop board -->
      <section
        class="rounded-lg border border-slate-200 bg-slate-50 p-3 col-span-2"
        @dragover.prevent
        @drop="drop('ready')"
      >
        <h3 class="font-semibold">Готовые видео · {{ completed.length }}</h3>
        <p class="mt-1 text-sm text-slate-500">Рендеры с вшитыми субтитрами.</p>
        <div class="grid grid-cols-2 gap-3 mt-3">
          <p v-if="!completed.length" class="text-sm text-slate-500">
            Пока пусто.
          </p>
          <article
            v-for="clip in completed"
            :key="clip.id"
            class="rounded bg-white p-3 shadow-sm"
          >
            <video
              v-if="videosByClip.get(clip.id)?.url"
              :src="videosByClip.get(clip.id)?.url"
              controls
              class="mb-3 aspect-[9/16] w-full rounded bg-black object-contain max-h-50"
            />
            <img
              v-else-if="clip.thumbnailURL"
              :src="clip.thumbnailURL"
              :alt="`Превью клипа: ${clip.title}`"
              class="mb-3 aspect-video w-full rounded object-cover"
            >
            <b>{{ clip.title }}</b>
            <p class="mt-1 text-sm text-slate-600">
              {{ clip.streamerName }} · Готово
            </p>
            <div
              v-if="videosByClip.get(clip.id)"
              class="mt-3 flex gap-3 text-violet-700"
            >
              <a :href="videosByClip.get(clip.id)?.twitchUrl" target="_blank"
                >Оригинал</a
              >
              <a :href="videosByClip.get(clip.id)?.url" target="_blank"
                >Открыть</a
              >
              <a :href="videosByClip.get(clip.id)?.url" download>Скачать</a>
            </div>
            <AppButton
              class="mt-3"
              variant="danger"
              :disabled="busy === clip.id"
              @click="remove(clip)"
            >
              Удалить
            </AppButton>
          </article>
        </div>
      </section>
    </div>

    <div
      v-if="processClipID"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="template-dialog-title"
    >
      <section
        class="max-h-[85vh] w-full max-w-3xl overflow-auto rounded-xl bg-white p-5 shadow-2xl"
      >
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 id="template-dialog-title" class="text-lg font-semibold">
              Выберите шаблон
            </h2>
            <p class="mt-1 text-sm text-slate-600">
              Снимок выбранного шаблона будет сохранён в задаче и не изменится
              после редактирования шаблона.
            </p>
          </div>
          <button
            type="button"
            class="rounded px-2 py-1 text-slate-600 hover:bg-slate-100"
            aria-label="Закрыть выбор шаблона"
            @click="processClipID = null"
          >
            ×
          </button>
        </div>
        <p v-if="templatesLoading" class="mt-4 text-slate-500">
          Загружаем шаблоны…
        </p>
        <div v-else class="mt-4 grid gap-3 sm:grid-cols-2">
          <button
            v-for="template in templates"
            :key="template.id"
            type="button"
            class="overflow-hidden rounded-lg border border-slate-200 text-left hover:border-violet-500 hover:ring-2 hover:ring-violet-100"
            @click="chooseTemplate(template.id)"
          >
            <img
              v-if="template.previewUrl"
              :src="template.previewUrl"
              :alt="`Превью шаблона ${template.name}`"
              class="aspect-video w-full object-cover"
            >
            <div
              v-else
              class="flex aspect-video items-center justify-center bg-gradient-to-br from-violet-950 to-slate-900 text-sm text-violet-100"
            >
              9:16 · {{ template.config.layers.length }} слоёв
            </div>
            <div class="p-3">
              <b>{{ template.name }}</b>
              <p class="mt-1 text-sm text-slate-600">
                {{ template.description || 'Без описания' }}
              </p>
            </div>
          </button>
        </div>
      </section>
    </div>
  </section>
</template>
