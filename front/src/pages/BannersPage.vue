<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppButton from '../shared/ui/AppButton.vue'
import EmptyState from '../shared/ui/EmptyState.vue'
import ErrorState from '../shared/ui/ErrorState.vue'

type B = { id: string; name: string; size: number }

const rows = ref<B[]>([]),
  file = ref<File | null>(null),
  error = ref('')

async function load() {
  const r = await fetch('/api/banners')
  rows.value = (await r.json()).data || []
}

async function upload() {
  if (!file.value) return

  const f = new FormData()
  f.append('file', file.value)
  const r = await fetch('/api/banners', { method: 'POST', body: f })

  if (!r.ok) {
    error.value = (await r.json()).error.message
    return
  }

  file.value = null
  await load()
}

async function remove(id: string) {
  await fetch(`/api/banners/${id}`, { method: 'DELETE' })
  load()
}

onMounted(load)
</script>

<template>
  <section>
    <h2 class="text-xl font-semibold">Баннеры</h2>
    <p class="mt-2 text-slate-600">Изображения хранятся в object storage.</p>
    <form class="my-6 flex gap-2" @submit.prevent="upload">
      <input
        type="file"
        accept="image/*"
        @change="file=($event.target as HTMLInputElement).files?.[0]||null"
      ><AppButton :disabled="!file">Загрузить</AppButton>
    </form>
    <ErrorState v-if="error" :message="error" />
    <EmptyState v-if="!rows.length" message="Баннеров пока нет." />
    <article
      v-for="b in rows"
      :key="b.id"
      class="mt-2 flex justify-between rounded bg-white p-4 shadow-sm"
    >
      <span>{{ b.name }} · {{ Math.round(b.size/1024) }} КБ</span
      ><AppButton variant="danger" @click="remove(b.id)">Удалить</AppButton>
    </article>
  </section>
</template>
