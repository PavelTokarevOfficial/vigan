<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import AppButton from '../shared/ui/AppButton.vue'
import EmptyState from '../shared/ui/EmptyState.vue'
import ErrorState from '../shared/ui/ErrorState.vue'
type Clip = {
  id: string
  title: string
  streamerName: string
  thumbnailURL: string
  status: string
  error: string
  currentStep: string
  progress: number
}
const clips = ref<Clip[]>([]),
  error = ref('')
async function load() {
  const clipResponse = await fetch('/api/clips')
  if (!clipResponse.ok) {
    error.value = 'Не удалось загрузить pipeline'
    return
  }
  clips.value = (await clipResponse.json()).data || []
}
async function action(id: string, path: string) {
  const r = await fetch(`/api/clips/${id}/${path}`, {
    method: 'POST',
  })
  if (!r.ok) {
    error.value =
      (await r.json()).error?.message || 'Не удалось поставить job в очередь'
  }
  await load()
}
let timer: number | undefined
onMounted(() => {
  load()
  timer = window.setInterval(load, 3000)
})
onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
})
</script>
<template>
  <section>
    <h2 class="text-xl font-semibold">Pipeline</h2>
    <p class="mt-2 text-slate-600">Импортированные клипы и фоновые jobs.</p>
    <ErrorState v-if="error" :message="error" />
    <EmptyState
      v-if="!clips.length"
      message="Импортированных клипов пока нет."
    />
    <article
      v-for="clip in clips"
      :key="clip.id"
      class="mt-3 flex gap-4 rounded bg-white p-4 shadow-sm"
    >
      <img
        v-if="clip.thumbnailURL"
        :src="clip.thumbnailURL"
        :alt="`Превью клипа: ${clip.title}`"
        class="h-20 w-36 rounded object-cover"
      >
      <div class="flex-1">
        <b>{{ clip.title }}</b>
        <p class="mt-1 text-sm text-slate-600">
          {{ clip.streamerName }} · статус: {{ clip.status
          }}<span v-if="clip.currentStep">
            · {{ clip.currentStep }} · {{ clip.progress }}%</span
          >
        </p>
        <p v-if="clip.error" class="text-sm text-red-600">{{ clip.error }}</p>
      </div>
      <AppButton
        v-if="clip.status === 'downloaded' || clip.status === 'ready_to_render'"
        @click="action(clip.id, 'process')"
      >
        Process</AppButton
      >
      <AppButton
        v-if="clip.status === 'failed'"
        variant="secondary"
        @click="action(clip.id, 'retry')"
        >Retry</AppButton
      >
    </article>
  </section>
</template>
