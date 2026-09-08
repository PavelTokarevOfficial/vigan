<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import ErrorState from '../shared/ui/ErrorState.vue'
import { readError } from '../shared/api/http'
import ClipImportButton from '../features/clip-import/ClipImportButton.vue'
import type { Streamer } from '../entities/streamer/model/types'
import type { TwitchClip } from '../entities/clip/model/types'

const streamers = ref<Streamer[]>([]),
  selected = ref(''),
  clips = ref<TwitchClip[]>([]),
  error = ref(''),
  busy = ref('')

function streamerName() {
  return streamers.value.find((s) => s.id === selected.value)?.displayName || ''
}

onMounted(async () => {
  const r = await fetch('/api/streamers')
  streamers.value = (await r.json()).data || []
})

watch(selected, async (id) => {
  if (!id) return

  error.value = ''
  const r = await fetch(`/api/streamers/${id}/clips`)

  if (!r.ok) {
    error.value = await readError(r, 'Не удалось получить Twitch clips')
    clips.value = []
    return
  }

  clips.value = (await r.json()).data || []
})

async function save(clip: TwitchClip) {
  busy.value = clip.id

  const r = await fetch('/api/clips/import', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ streamerId: selected.value, clip }),
  })

  if (!r.ok) error.value = await readError(r, 'Не удалось сохранить clip')
  else clip.saved = true
  busy.value = ''
}
</script>

<template>
  <section>
    <h2 class="text-xl font-semibold">Клипы Twitch</h2>
    <select v-model="selected" class="mt-5">
      <option value="">Выберите стримера</option>
      <option v-for="s in streamers" :key="s.id" :value="s.id">
        {{ s.displayName }}
      </option>
    </select><ErrorState v-if="error" :message="error" />
    <div class="mt-6 grid gap-4 md:grid-cols-3">
      <article
        v-for="c in clips"
        :key="c.id"
        class="overflow-hidden rounded bg-white shadow-sm"
      >
        <img
          :src="c.thumbnail_url"
          :alt="`Превью клипа: ${c.title}`"
          class="aspect-video w-full object-cover"
        >
        <div class="p-3">
          <b>{{ c.title }}</b>
          <p class="my-2 text-sm text-slate-600">
            {{ streamerName() }} ·
            {{ new Date(c.created_at).toLocaleDateString() }} ·
            {{ Math.round(c.duration) }} сек.
          </p>
          <div class="flex justify-between">
            <a :href="c.url" target="_blank" class="text-violet-700">Twitch</a>
            <ClipImportButton
              :saved="c.saved"
              :busy="busy===c.id"
              @save="save(c)"
            />
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
