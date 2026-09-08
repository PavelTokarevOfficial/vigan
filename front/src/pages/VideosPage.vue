<script setup lang="ts">
import { onMounted, ref } from 'vue'

type V = {
  clipID: string
  title: string
  streamer: string
  twitchUrl: string
  storageKey: string
  createdAt: string
  url: string
}

const videos = ref<V[]>([])

onMounted(async () => {
  const r = await fetch('/api/videos')
  videos.value = (await r.json()).data || []
})
</script>

<template>
  <section>
    <h2 class="text-xl font-semibold">Готовые видео</h2>

    <p class="mt-2 text-slate-600">Vertical renders из object storage.</p>

    <div v-if="!videos.length" class="mt-6 rounded border border-dashed p-8 text-slate-500">
      Готовых видео пока нет.
    </div>

    <div class='grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3'>



      <article v-for="v in videos" :key="v.storageKey" class="mt-4 rounded bg-white p-4 shadow-sm">
        <video :src="v.url" controls class="max-h-96 w-full bg-black" />

        <div class="mt-2 flex flex-wrap justify-between gap-2">
          <span>
            <b>{{ v.title }}</b>
            ·
            {{ v.streamer }}
            ·
            {{ new Date(v.createdAt).toLocaleString() }}
          </span>

          <span class="flex gap-3">
            <a :href="v.twitchUrl" target="_blank" class="text-violet-700">Оригинал</a><a :href="v.url" target="_blank"
              class="text-violet-700">Открыть</a><a :href="v.url" download class="text-violet-700">Скачать</a></span>
        </div>
      </article>
    </div>
  </section>
</template>
