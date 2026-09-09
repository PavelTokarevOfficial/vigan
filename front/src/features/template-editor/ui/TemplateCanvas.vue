<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { Layer, TemplateConfig } from '@/entities/template/model/types'

const props = defineProps<{
  config: TemplateConfig
  selectedLayerId: string | null
}>()
const emit = defineEmits<{
  select: [id: string]
  update: [id: string, patch: Partial<Layer>]
}>()

const scale = 0.3
const stageRef = ref()
const transformerRef = ref()

function shape(layer: Layer) {
  return {
    id: `layer-${layer.id}`,
    x: layer.x * scale,
    y: layer.y * scale,
    width: Math.max(layer.width * scale, 1),
    height: Math.max(layer.height * scale, 1),
    opacity: layer.opacity,
    draggable: true,
  }
}

function label(layer: Layer) {
  if (layer.type === 'subtitles') return 'Субтитры: пример текста'
  if (layer.type === 'input_video') return layer.name || 'Исходный клип'
  if (layer.type === 'text') return layer.text || 'Текст'
  if (layer.type === 'audio') return 'Аудио'
  return layer.name
}

function color(layer: Layer) {
  if (layer.type === 'color') return layer.color || '#111827'
  if (layer.type === 'subtitles') return '#18181b'
  if (layer.type === 'text') return '#312e81'
  if (layer.type === 'audio') return '#0f766e'
  if (layer.type === 'input_video') return '#4c1d95'
  return '#1d4ed8'
}

function updateTransformer() {
  void nextTick(() => {
    const stage = stageRef.value?.getNode?.()
    const transformer = transformerRef.value?.getNode?.()
    if (!stage || !transformer) return
    const node = props.selectedLayerId
      ? stage.findOne(`#layer-${props.selectedLayerId}`)
      : null
    transformer.nodes(node ? [node] : [])
    transformer.getLayer()?.batchDraw()
  })
}

function dragged(
  layer: Layer,
  event: { target: { x: () => number; y: () => number } },
) {
  emit('update', layer.id, {
    x: Math.round(event.target.x() / scale),
    y: Math.round(event.target.y() / scale),
  })
}

function transformed(
  layer: Layer,
  event: {
    target: {
      x: () => number
      y: () => number
      width: () => number
      height: () => number
      scaleX: (value?: number) => number
      scaleY: (value?: number) => number
    }
  },
) {
  const node = event.target
  const width = Math.max(1, Math.round((node.width() * node.scaleX()) / scale))
  const height = Math.max(
    1,
    Math.round((node.height() * node.scaleY()) / scale),
  )
  node.scaleX(1)
  node.scaleY(1)
  emit('update', layer.id, {
    x: Math.round(node.x() / scale),
    y: Math.round(node.y() / scale),
    width,
    height,
  })
}

watch(() => [props.selectedLayerId, props.config.layers], updateTransformer, {
  deep: true,
})
</script>

<template>
  <div
    class="overflow-auto rounded-xl border border-slate-200 bg-slate-950 p-4"
  >
    <v-stage
      ref="stageRef"
      :config="{ width: config.canvas.width * scale, height: config.canvas.height * scale }"
      class="mx-auto shadow-2xl"
      @mousedown="updateTransformer"
    >
      <v-layer>
        <v-rect
          :config="{
            x: 0,
            y: 0,
            width: config.canvas.width * scale,
            height: config.canvas.height * scale,
            fill: config.canvas.background,
          }"
        />
        <template v-for="layer in config.layers" :key="layer.id">
          <v-group
            v-if="layer.visible && layer.type !== 'audio'"
            :config="shape(layer)"
            @click="emit('select', layer.id)"
            @tap="emit('select', layer.id)"
            @dragend="dragged(layer, $event)"
            @transformend="transformed(layer, $event)"
          >
            <v-rect
              :config="{
                width: Math.max(layer.width * scale, 1),
                height: Math.max(layer.height * scale, 1),
                fill: color(layer),
                stroke: selectedLayerId === layer.id ? '#fbbf24' : '#cbd5e1',
                strokeWidth: selectedLayerId === layer.id ? 2 : 1,
                cornerRadius: 4,
              }"
            />
            <v-text
              :config="{
                text: label(layer),
                width: Math.max(layer.width * scale - 12, 1),
                height: Math.max(layer.height * scale - 12, 1),
                x: 6,
                y: 6,
                fontSize: Math.min(16, Math.max(10, layer.width * scale * 0.07)),
                fill: '#fff',
                align: 'center',
                verticalAlign: 'middle',
                wrap: 'word',
              }"
            />
          </v-group>
        </template>
        <v-transformer
          ref="transformerRef"
          :config="{ rotateEnabled: false, keepRatio: false, borderStroke: '#fbbf24' }"
        />
      </v-layer>
    </v-stage>
  </div>
</template>
