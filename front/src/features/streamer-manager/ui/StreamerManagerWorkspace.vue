<script setup lang="ts">
import { Pencil, Plus, Trash2 } from '@lucide/vue'
import { onMounted } from 'vue'
import { useStreamerManager } from '@/features/streamer-manager/model/useStreamerManager'
import AppButton from '@/shared/ui/AppButton.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'
import AddStreamersDialog from './AddStreamersDialog.vue'

const manager = useStreamerManager()
const { busy, editNickname, editing, error, notice, rows } = manager

function updatePriority(event: Event, id: string) {
  const priority = Number((event.target as HTMLInputElement).value)
  const streamer = rows.value.find((item) => item.id === id)
  if (streamer) void manager.setPriority(streamer, priority)
}

onMounted(() => void manager.load())
</script>

<template>
  <section>
    <div class="mb-10 flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-4xl font-semibold">Стримеры</h1>
      <AppButton type="button" @click="manager.openAddDialog"
        ><Plus class="mr-1 inline size-4" />Добавить</AppButton
      >
    </div>
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
        @submit.prevent="manager.saveEdit"
      >
        <input v-model="editNickname" required>
        <AppButton type="submit" :disabled="busy">Сохранить</AppButton>
        <AppButton type="button" variant="secondary" @click="manager.cancelEdit"
          >Отмена</AppButton
        >
      </form>
      <div v-else class="flex items-center justify-between gap-3">
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
        <div class="flex shrink-0 gap-2">
          <label class="text-sm text-slate-600"
            >Приоритет<input
              type="number"
              min="0"
              step="1"
              class="ml-2 w-18 rounded border border-slate-300 px-2 py-1 text-right text-slate-900"
              :value="row.priority"
              :disabled="busy"
              @change="updatePriority($event, row.id)"
            ></label
          >
          <AppButton
            variant="secondary"
            :disabled="busy"
            @click="manager.startEdit(row)"
            ><Pencil class="mr-1 inline size-4" />Изменить</AppButton
          >
          <AppButton
            variant="danger"
            :disabled="busy"
            @click="manager.remove(row)"
            ><Trash2 class="mr-1 inline size-4" />Удалить</AppButton
          >
        </div>
      </div>
    </article>
    <AddStreamersDialog :manager="manager" />
  </section>
</template>
