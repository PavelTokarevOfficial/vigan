<script setup lang="ts">
import { Copy, Pencil, Plus, Trash2 } from '@lucide/vue'
import { onMounted, ref } from 'vue'
import type { VideoTemplate } from '@/entities/template/model/types'
import { readData, readError } from '@/shared/api/http'
import AppButton from '@/shared/ui/AppButton.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'

const templates = ref<VideoTemplate[]>([])
const error = ref('')
const busy = ref('')

async function load() {
  try {
    templates.value = await readData<VideoTemplate[]>(
      await fetch('/api/templates'),
    )
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось загрузить шаблоны'
  }
}
async function duplicate(id: string) {
  busy.value = id
  error.value = ''
  const response = await fetch(`/api/templates/${id}/duplicate`, {
    method: 'POST',
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось создать копию')
  busy.value = ''
  await load()
}
async function remove(item: VideoTemplate) {
  if (!window.confirm(`Удалить шаблон «${item.name}»?`)) return
  busy.value = item.id
  error.value = ''
  const response = await fetch(`/api/templates/${item.id}`, {
    method: 'DELETE',
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось удалить шаблон')
  busy.value = ''
  await load()
}

onMounted(() => void load())
</script>

<template>
  <section>
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-xl font-semibold">Шаблоны видео</h1>
        <p class="mt-1 text-slate-600">
          Композиции для рендера: слои, субтитры, фоны и ассеты.
        </p>
      </div>
      <RouterLink to="/templates/new"
        ><AppButton
          ><Plus class="mr-1 inline size-4" />Новый шаблон</AppButton
        ></RouterLink
      >
    </div>
    <ErrorState v-if="error" :message="error" />
    <EmptyState
      v-if="!templates.length"
      class="mt-6"
      message="Шаблонов пока нет."
    />
    <div v-else class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="item in templates"
        :key="item.id"
        class="overflow-hidden rounded-xl border border-slate-200 bg-white"
      >
        <img
          v-if="item.previewUrl && item.previewAssetId"
          :src="item.previewUrl"
          :alt="`Превью шаблона ${item.name}`"
          class="aspect-video w-full object-cover"
        >
        <div
          v-else
          class="flex aspect-video items-center justify-center bg-gradient-to-br from-violet-950 to-slate-900 text-sm text-violet-100"
        >
          9:16 · {{ item.config.layers.length }} слоёв
        </div>
        <div class="p-4">
          <h2 class="font-semibold">{{ item.name }}</h2>
          <p class="mt-1 min-h-10 text-sm text-slate-600">
            {{ item.description || 'Без описания' }}
          </p>
          <div class="mt-4 flex flex-wrap gap-2">
            <RouterLink :to="`/templates/${item.id}`"
              ><AppButton variant="secondary"
                ><Pencil class="mr-1 inline size-4" />Открыть</AppButton
              ></RouterLink
            >
            <AppButton
              variant="secondary"
              :disabled="busy === item.id"
              @click="duplicate(item.id)"
              ><Copy class="mr-1 inline size-4" />Копия</AppButton
            >
            <AppButton
              variant="danger"
              :disabled="busy === item.id"
              @click="remove(item)"
              ><Trash2 class="mr-1 inline size-4" />Удалить</AppButton
            >
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
