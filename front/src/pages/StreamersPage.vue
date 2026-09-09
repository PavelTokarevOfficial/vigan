<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppButton from '../shared/ui/AppButton.vue'
import EmptyState from '../shared/ui/EmptyState.vue'
import ErrorState from '../shared/ui/ErrorState.vue'
import { readData, readError } from '../shared/api/http'
import type { Streamer } from '../entities/streamer/model/types'

const rows = ref<Streamer[]>([]),
  nicknames = ref(''),
  error = ref(''),
  notice = ref(''),
  editing = ref<Streamer | null>(null),
  editNickname = ref('')

async function load() {
  try {
    rows.value = await readData<Streamer[]>(await fetch('/api/streamers'))
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Не удалось загрузить список'
  }
}

async function add() {
  error.value = ''
  notice.value = ''

  const r = await fetch('/api/streamers/bulk', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ twitchLogins: nicknames.value.split(/\r?\n/) }),
  })

  if (!r.ok) {
    error.value = await readError(r, 'Не удалось добавить стримера')
    return
  }

  const data = await r.json()
  notice.value = data.data.created
    ? `Добавлено стримеров: ${data.data.created}.`
    : 'Все указанные стримеры уже были добавлены.'
  nicknames.value = ''
  await load()
}

async function remove(id: string) {
  await fetch(`/api/streamers/${id}`, { method: 'DELETE' })
  load()
}

function startEdit(row: Streamer) {
  editing.value = row
  editNickname.value = row.twitchLogin
}

async function saveEdit() {
  if (!editing.value) return

  const r = await fetch(`/api/streamers/${editing.value.id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      twitchLogin: editNickname.value,
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
    <p class="mt-1 text-slate-600">
      Добавьте один или несколько ников Twitch: каждый ник с новой строки.
    </p>
    <form class="my-6 flex max-w-xl flex-col gap-2" @submit.prevent="add">
      <textarea
        v-model="nicknames"
        class='border border-violet-600 rounded-xl p-3'
        rows="8"
        placeholder="chocokokko_&#10;KaiCenat&#10;xqc"
        required
      />
      <AppButton type="submit">Добавить</AppButton>
    </form>
    <ErrorState v-if="error" :message="error" />
    <p v-if="notice" class="mb-4 text-sm text-emerald-700">{{ notice }}</p>
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
        <input v-model="editNickname" required>
        <AppButton type="submit">Сохранить</AppButton>
        <AppButton type="button" variant="secondary" @click="editing = null"
          >Отмена</AppButton
        >
      </form>
      <div v-else class="flex items-center justify-between">
        <div>
          <b>{{ row.displayName }}</b>
          <span
            v-if="row.displayName !== row.twitchLogin"
            class="ml-2 text-slate-500"
            >{{
              row.twitchLogin
            }}</span
          >
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
