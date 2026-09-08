<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppButton from '../shared/ui/AppButton.vue'
import EmptyState from '../shared/ui/EmptyState.vue'
import ErrorState from '../shared/ui/ErrorState.vue'
import { readData, readError } from '../shared/api/http'
import type { Streamer } from '../entities/streamer/model/types'

const rows = ref<Streamer[]>([]),
  login = ref(''),
  name = ref(''),
  error = ref(''),
  editing = ref<Streamer | null>(null),
  editLogin = ref(''),
  editName = ref('')

async function load() {
  try {
    rows.value = await readData<Streamer[]>(await fetch('/api/streamers'))
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Не удалось загрузить список'
  }
}

async function add() {
  error.value = ''

  const r = await fetch('/api/streamers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ twitchLogin: login.value, displayName: name.value }),
  })

  if (!r.ok) {
    error.value = await readError(r, 'Не удалось добавить стримера')
    return
  }

  login.value = ''
  name.value = ''
  load()
}

async function remove(id: string) {
  await fetch(`/api/streamers/${id}`, { method: 'DELETE' })
  load()
}

function startEdit(row: Streamer) {
  editing.value = row
  editLogin.value = row.twitchLogin
  editName.value = row.displayName
}

async function saveEdit() {
  if (!editing.value) return

  const r = await fetch(`/api/streamers/${editing.value.id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      twitchLogin: editLogin.value,
      displayName: editName.value,
    }),
  })

  if (!r.ok) {
    error.value = await readError(r, 'Не удалось изменить стримера')
    return
  }

  editing.value = null
  await load()
}

onMounted(load)
</script>
<template>
  <section>
    <h2 class="text-xl font-semibold">Стримеры</h2>
    <p class="mt-1 text-slate-600">Добавьте Twitch login для поиска клипов.</p>
    <form class="my-6 flex gap-2" @submit.prevent="add">
      <input v-model="login" placeholder="twitch login" required>
      <input v-model="name" placeholder="Отображаемое имя" required>
      <AppButton>Добавить</AppButton>
    </form>
    <ErrorState v-if="error" :message="error" />
    <EmptyState v-if="!rows.length" message="Стримеров пока нет." />
    <article
      v-for="row in rows"
      :key="row.id"
      class="mb-2 rounded bg-white p-4 shadow-sm"
    >
      <form
        v-if="editing?.id === row.id"
        class="flex flex-wrap gap-2"
        @submit.prevent="saveEdit"
      >
        <input v-model="editLogin" required><input v-model="editName" required>
        <AppButton>Сохранить</AppButton>
        <AppButton type="button" variant="secondary" @click="editing = null"
          >Отмена</AppButton
        >
      </form>
      <div v-else class="flex items-center justify-between">
        <div>
          <b>{{ row.displayName }}</b
          ><span class="ml-2 text-slate-500">{{ row.twitchLogin }}</span>
        </div>
        <div class="flex gap-2">
          <AppButton variant="secondary" @click="startEdit(row)"
            >Изменить</AppButton
          >
          <AppButton variant="danger" @click="remove(row.id)"
            >Удалить</AppButton
          >
        </div>
      </div>
    </article>
  </section>
</template>
