<script setup lang="ts">
import { Upload } from '@lucide/vue'
import { ref } from 'vue'

defineProps<{ busy: boolean }>()
const emit = defineEmits<{ files: [files: FileList] }>()
const dragActive = ref(false)

function selectFiles(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.files?.length) emit('files', input.files)
  input.value = ''
}
function dropFiles(event: DragEvent) {
  dragActive.value = false
  if (event.dataTransfer?.files.length) emit('files', event.dataTransfer.files)
}
</script>

<template>
  <label
    class="mt-4 flex min-h-48 w-full cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed p-6 text-center transition"
    :class="
      dragActive
        ? 'border-violet-500 bg-violet-50 text-violet-800'
        : 'border-slate-300 bg-white text-slate-600 hover:border-violet-400 hover:bg-violet-50/50'
    "
    @dragenter.prevent="dragActive = true"
    @dragover.prevent="dragActive = true"
    @dragleave="dragActive = false"
    @drop.prevent="dropFiles"
  >
    <input
      type="file"
      multiple
      accept="image/*,video/*,audio/*"
      class="sr-only"
      :disabled="busy"
      @change="selectFiles"
    >
    <Upload class="size-8" />
    <span class="mt-3 font-medium">Нажмите на блок или перетащите файлы</span>
    <span class="mt-1 text-sm text-slate-500">
      Изображения, GIF, видео и аудио
    </span>
    <span v-if="busy" class="mt-3 text-sm font-medium">Загрузка файлов…</span>
  </label>
</template>
