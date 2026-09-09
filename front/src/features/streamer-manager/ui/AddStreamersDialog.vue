<script setup lang="ts">
import type { StreamerManager } from '@/features/streamer-manager/model/useStreamerManager'
import AppButton from '@/shared/ui/AppButton.vue'

defineProps<{ manager: StreamerManager }>()
</script>

<template>
  <div
    v-if="manager.addDialogOpen.value"
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby="add-streamers-title"
  >
    <form
      class="w-full max-w-xl rounded-xl bg-white p-5 shadow-2xl"
      @submit.prevent="manager.add"
    >
      <h2 id="add-streamers-title" class="text-lg font-semibold">
        Добавить стримеров
      </h2>
      <p class="mt-1 text-sm text-slate-600">
        Укажите один Twitch-ник в каждой строке. За раз можно добавить до 100.
      </p>
      <textarea
        v-model="manager.nicknames.value"
        class="mt-4 min-h-48 w-full rounded-xl border border-violet-600 p-3"
        rows="8"
        placeholder="chocokokko_&#10;KaiCenat&#10;xqc"
        required
      />
      <div class="mt-5 flex justify-end gap-2">
        <AppButton
          type="button"
          variant="secondary"
          @click="manager.closeAddDialog"
          >Отмена</AppButton
        ><AppButton
          type="submit"
          :disabled="manager.busy.value || !manager.nicknames.value.trim()"
          >Добавить</AppButton
        >
      </div>
    </form>
  </div>
</template>
