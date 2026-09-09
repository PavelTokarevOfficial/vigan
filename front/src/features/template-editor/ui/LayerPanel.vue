<script setup lang="ts">
import { Eye, EyeOff, Layers, Trash2 } from '@lucide/vue'
import type { Layer, LayerType } from '@/entities/template/model/types'

defineProps<{ layers: Layer[]; selectedLayerId: string | null }>()
const emit = defineEmits<{
  select: [id: string]
  add: [type: LayerType]
  update: [id: string, patch: Partial<Layer>]
  duplicate: [id: string]
  remove: [id: string]
  move: [id: string, direction: -1 | 1]
}>()

const addable: { type: LayerType; label: string }[] = [
  { type: 'input_video', label: 'Исходный клип' },
  { type: 'asset_video', label: 'Видео' },
  { type: 'image', label: 'Изображение' },
  { type: 'gif', label: 'GIF' },
  { type: 'subtitles', label: 'Субтитры' },
  { type: 'text', label: 'Текст' },
  { type: 'audio', label: 'Аудио' },
  { type: 'color', label: 'Цвет' },
]
</script>

<template>
  <aside class="rounded-xl border border-slate-200 bg-white p-3">
    <div class="flex items-center justify-between gap-2">
      <h2 class="flex items-center gap-2 font-semibold">
        <Layers class="size-4" />
        Слои
      </h2>
      <select
        class="max-w-34 text-sm"
        @change="emit('add', ($event.target as HTMLSelectElement).value as LayerType)"
      >
        <option value="" selected disabled>Добавить</option>
        <option v-for="item in addable" :key="item.type" :value="item.type">
          {{ item.label }}
        </option>
      </select>
    </div>
    <p class="mt-2 text-xs text-slate-500">Нижний слой рисуется первым.</p>
    <ol class="mt-3 space-y-2">
      <li
        v-for="(layer, index) in layers"
        :key="layer.id"
        class="rounded border p-2"
        :class="selectedLayerId === layer.id ? 'border-violet-500 bg-violet-50' : 'border-slate-200'"
      >
        <button
          type="button"
          class="flex w-full items-center justify-between gap-2 text-left"
          @click="emit('select', layer.id)"
        >
          <span class="min-w-0 truncate font-medium">{{ layer.name }}</span>
          <span class="shrink-0 text-xs text-slate-500">{{ layer.type }}</span>
        </button>
        <div class="mt-2 flex items-center gap-1 text-xs">
          <button
            type="button"
            class="rounded p-1 hover:bg-white"
            :aria-label="layer.visible ? 'Скрыть слой' : 'Показать слой'"
            @click="emit('update', layer.id, { visible: !layer.visible })"
          >
            <Eye v-if="layer.visible" class="size-4" />
            <EyeOff v-else class="size-4" />
          </button>
          <button
            type="button"
            class="rounded px-1 hover:bg-white"
            :disabled="index === 0"
            @click="emit('move', layer.id, -1)"
          >
            ↓
          </button>
          <button
            type="button"
            class="rounded px-1 hover:bg-white"
            :disabled="index === layers.length - 1"
            @click="emit('move', layer.id, 1)"
          >
            ↑
          </button>
          <button
            type="button"
            class="rounded px-1 hover:bg-white"
            @click="emit('duplicate', layer.id)"
          >
            Копия
          </button>
          <button
            type="button"
            class="ml-auto rounded p-1 text-red-600 hover:bg-red-50"
            aria-label="Удалить слой"
            @click="emit('remove', layer.id)"
          >
            <Trash2 class="size-4" />
          </button>
        </div>
      </li>
    </ol>
  </aside>
</template>
