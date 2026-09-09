<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import AppButton from '../shared/ui/AppButton.vue'
import ErrorState from '../shared/ui/ErrorState.vue'
import { readError } from '../shared/api/http'

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
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось загрузить доску'
  }
}

async function action(id: string, path: 'download' | 'process' | 'retry') {
  busy.value = id
  error.value = ''
  const response = await fetch(`/api/clips/${id}/${path}`, { method: 'POST' })
  if (!response.ok) {
    error.value = await readError(
      response,
      'Не удалось поставить задачу в очередь',
    )
  }
  busy.value = ''
  await load()
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
  return ['saved', 'downloaded', 'failed'].includes(clip.status)
}

function drop(column: Column) {
  const clip = dragged.value
  dragged.value = null
  if (!clip) return

  if (column === 'downloaded' && clip.status === 'saved') {
    void action(clip.id, 'download')
  }
  if (column === 'ready' && clip.status === 'downloaded') {
    void action(clip.id, 'process')
  }
}

function statusText(clip: Clip) {
  if (clip.status === 'saved') return 'В избранном'
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
                v-if="clip.status === 'downloaded'"
                :disabled="busy === clip.id"
                @click="action(clip.id, 'process')"
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
  </section>
</template>
