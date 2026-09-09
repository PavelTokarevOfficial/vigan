import { computed, ref } from 'vue'
import type {
  Layer,
  LayerType,
  TemplateConfig,
} from '@/entities/template/model/types'

// Template config is deliberately JSON-only. JSON cloning also unwraps Vue's
// reactive Proxy objects before putting a snapshot into undo/redo history.
const copy = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T

export function useTemplateEditor(initial: TemplateConfig) {
  const draft = ref(copy(initial))
  const selectedLayerID = ref<string | null>(
    draft.value.layers.at(-1)?.id ?? null,
  )
  const history = ref<TemplateConfig[]>([copy(initial)])
  const historyIndex = ref(0)
  const selectedLayer = computed(
    () =>
      draft.value.layers.find((layer) => layer.id === selectedLayerID.value) ??
      null,
  )
  const isDirty = computed(
    () => JSON.stringify(draft.value) !== JSON.stringify(history.value[0]),
  )
  const canUndo = computed(() => historyIndex.value > 0)
  const canRedo = computed(() => historyIndex.value < history.value.length - 1)

  function apply(next: TemplateConfig) {
    const snapshot = copy(next)
    history.value = history.value.slice(0, historyIndex.value + 1)
    history.value.push(snapshot)
    historyIndex.value = history.value.length - 1
    draft.value = copy(snapshot)
  }

  function updateLayer(id: string, patch: Partial<Layer>) {
    apply({
      ...draft.value,
      layers: draft.value.layers.map((layer) =>
        layer.id === id ? { ...layer, ...patch } : layer,
      ),
    })
  }

  function addLayer(type: LayerType) {
    const id = `${type}-${crypto.randomUUID().slice(0, 8)}`
    const layer: Layer = {
      id,
      name: layerName(type),
      type,
      x: 100,
      y: 100,
      width: type === 'audio' ? 0 : 600,
      height: type === 'audio' ? 0 : 300,
      visible: true,
      opacity: 1,
      fit: 'contain',
      ...(type === 'text' ? { text: 'Новый текст' } : {}),
      ...(type === 'color' ? { color: '#111827' } : {}),
      ...(type === 'subtitles'
        ? {
            style: {
              fontSize: 8,
              alignment: 2,
              marginV: 100,
              outline: 2,
              primaryColor: '&H00FFFFFF',
              outlineColor: '&H00000000',
            },
          }
        : {}),
    }
    apply({ ...draft.value, layers: [...draft.value.layers, layer] })
    selectedLayerID.value = id
  }

  function removeLayer(id: string) {
    apply({
      ...draft.value,
      layers: draft.value.layers.filter((layer) => layer.id !== id),
    })
    selectedLayerID.value = draft.value.layers.at(-1)?.id ?? null
  }

  function duplicateLayer(id: string) {
    const source = draft.value.layers.find((layer) => layer.id === id)
    if (!source) return
    const duplicate = {
      ...copy(source),
      id: `${source.type}-${crypto.randomUUID().slice(0, 8)}`,
      name: `${source.name} — копия`,
      x: source.x + 30,
      y: source.y + 30,
    }
    apply({ ...draft.value, layers: [...draft.value.layers, duplicate] })
    selectedLayerID.value = duplicate.id
  }

  function moveLayer(id: string, direction: -1 | 1) {
    const index = draft.value.layers.findIndex((layer) => layer.id === id)
    const nextIndex = index + direction
    if (index < 0 || nextIndex < 0 || nextIndex >= draft.value.layers.length)
      return
    const layers = [...draft.value.layers]
    ;[layers[index], layers[nextIndex]] = [layers[nextIndex], layers[index]]
    apply({ ...draft.value, layers })
  }

  function undo() {
    if (!canUndo.value) return
    historyIndex.value -= 1
    draft.value = copy(history.value[historyIndex.value])
  }
  function redo() {
    if (!canRedo.value) return
    historyIndex.value += 1
    draft.value = copy(history.value[historyIndex.value])
  }
  function markSaved() {
    history.value = [copy(draft.value)]
    historyIndex.value = 0
  }
  function replace(config: TemplateConfig) {
    draft.value = copy(config)
    history.value = [copy(config)]
    historyIndex.value = 0
    selectedLayerID.value = config.layers.at(-1)?.id ?? null
  }

  return {
    draft,
    selectedLayerID,
    selectedLayer,
    isDirty,
    canUndo,
    canRedo,
    updateLayer,
    addLayer,
    removeLayer,
    duplicateLayer,
    moveLayer,
    undo,
    redo,
    markSaved,
    replace,
  }
}

function layerName(type: LayerType) {
  return {
    input_video: 'Исходный клип',
    asset_video: 'Видео-ассет',
    image: 'Изображение',
    gif: 'GIF',
    subtitles: 'Субтитры',
    text: 'Текст',
    audio: 'Аудио',
    color: 'Цветной фон',
  }[type]
}
