<script setup lang="ts">
import { computed } from 'vue'
import type { Asset } from '@/entities/asset/model/types'
import type { Layer } from '@/entities/template/model/types'

const props = defineProps<{ layer: Layer | null; assets: Asset[] }>()
const emit = defineEmits<{ update: [patch: Partial<Layer>] }>()

const isAssetLayer = computed(() =>
  ['asset_video', 'image', 'gif', 'audio'].includes(props.layer?.type ?? ''),
)
const compatibleAssets = computed(() => {
  const type = props.layer?.type
  if (type === 'asset_video')
    return props.assets.filter((asset) => asset.kind === 'video')
  if (type === 'image')
    return props.assets.filter((asset) => asset.kind === 'image')
  if (type === 'gif')
    return props.assets.filter((asset) => asset.kind === 'gif')
  if (type === 'audio')
    return props.assets.filter((asset) => asset.kind === 'audio')
  return []
})
function numberValue(event: Event) {
  return Number((event.target as HTMLInputElement).value)
}
</script>

<template>
  <aside class="rounded-xl border border-slate-200 bg-white p-3">
    <h2 class="font-semibold">Свойства</h2>
    <p v-if="!layer" class="mt-3 text-sm text-slate-500">
      Выберите слой на canvas или в списке.
    </p>
    <div v-else class="mt-3 space-y-3">
      <label class="block text-sm"
        >Название<input
          class="mt-1 w-full"
          :value="layer.name"
          @change="emit('update', { name: ($event.target as HTMLInputElement).value })"
        ></label
      >
      <label v-if="layer.type === 'text'" class="block text-sm"
        >Текст<textarea
          class="mt-1 w-full rounded border border-slate-300 px-3 py-2"
          :value="layer.text"
          @change="emit('update', { text: ($event.target as HTMLTextAreaElement).value })"
        /></label
      >
      <label v-if="layer.type === 'color'" class="block text-sm"
        >Цвет<input
          class="mt-1 h-10 w-full"
          type="color"
          :value="layer.color || '#111827'"
          @change="emit('update', { color: ($event.target as HTMLInputElement).value })"
        ></label
      >
      <div
        v-if="isAssetLayer"
        class="rounded-lg border border-violet-200 bg-violet-50 p-2"
      >
        <p class="text-sm font-medium">Источник слоя</p>
        <p v-if="!compatibleAssets.length" class="mt-1 text-xs text-slate-600">
          Подходящих файлов нет. Сначала загрузите {{ layer.type }} в разделе
          «Ассеты».
        </p>
        <div v-else class="mt-2 grid gap-2">
          <button
            v-for="asset in compatibleAssets"
            :key="asset.id"
            type="button"
            class="flex items-center gap-2 rounded border bg-white p-2 text-left text-sm hover:border-violet-500"
            :class="layer.assetId === asset.id ? 'border-violet-600 ring-1 ring-violet-500' : 'border-slate-200'"
            @click="emit('update', { assetId: asset.id })"
          >
            <img
              v-if="asset.kind === 'image' || asset.kind === 'gif'"
              :src="asset.url"
              :alt="asset.name"
              class="size-9 rounded object-cover"
            >
            <span class="min-w-0 truncate">{{ asset.name }}</span>
          </button>
        </div>
        <label class="mt-2 block text-sm"
          >Или выберите из списка<select
            class="mt-1 w-full"
            :value="layer.assetId"
            @change="emit('update', { assetId: ($event.target as HTMLSelectElement).value })"
          >
            <option value="">Выберите файл</option>
            <option
              v-for="asset in compatibleAssets"
              :key="asset.id"
              :value="asset.id"
            >
              {{ asset.name }} · {{ asset.kind }}
            </option>
          </select></label
        >
      </div>
      <div v-if="layer.type !== 'audio'" class="grid grid-cols-2 gap-2">
        <label class="text-sm"
          >X<input
            class="mt-1 w-full"
            type="number"
            :value="layer.x"
            @change="emit('update', { x: numberValue($event) })"
          ></label
        >
        <label class="text-sm"
          >Y<input
            class="mt-1 w-full"
            type="number"
            :value="layer.y"
            @change="emit('update', { y: numberValue($event) })"
          ></label
        >
        <label class="text-sm"
          >Ширина<input
            class="mt-1 w-full"
            type="number"
            min="1"
            :value="layer.width"
            @change="emit('update', { width: numberValue($event) })"
          ></label
        >
        <label class="text-sm"
          >Высота<input
            class="mt-1 w-full"
            type="number"
            min="1"
            :value="layer.height"
            @change="emit('update', { height: numberValue($event) })"
          ></label
        >
      </div>
      <label class="block text-sm"
        >Прозрачность {{ Math.round(layer.opacity * 100) }}%<input
          class="mt-1 w-full"
          type="range"
          min="0"
          max="1"
          step="0.05"
          :value="layer.opacity"
          @input="emit('update', { opacity: numberValue($event) })"
        ></label
      >
      <label
        v-if="layer.type === 'input_video' || layer.type === 'asset_video' || layer.type === 'image' || layer.type === 'gif'"
        class="block text-sm"
        >Вписывание<select
          class="mt-1 w-full"
          :value="layer.fit || 'contain'"
          @change="emit('update', { fit: ($event.target as HTMLSelectElement).value as Layer['fit'] })"
        >
          <option value="contain">Вписать</option>
          <option value="cover">Заполнить</option>
          <option value="stretch">Растянуть</option>
        </select></label
      >
      <template v-if="layer.type === 'subtitles'">
        <label class="block text-sm"
          >Размер шрифта FFmpeg<input
            class="mt-1 w-full"
            type="number"
            min="1"
            :value="layer.style?.fontSize || 8"
            @change="emit('update', { style: { ...layer.style, fontSize: numberValue($event) } })"
          ></label
        >
        <label class="block text-sm"
          >Контур<input
            class="mt-1 w-full"
            type="number"
            min="0"
            :value="layer.style?.outline || 2"
            @change="emit('update', { style: { ...layer.style, outline: numberValue($event) } })"
          ></label
        >
      </template>
    </div>
  </aside>
</template>
