<script setup lang="ts">
import { ArrowLeft, Redo2, Save, Undo2 } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import type { Asset } from '@/entities/asset/model/types'
import {
  createDefaultConfig,
  type VideoTemplate,
} from '@/entities/template/model/types'
import { useTemplateEditor } from '@/features/template-editor/model/useTemplateEditor'
import LayerPanel from '@/features/template-editor/ui/LayerPanel.vue'
import PropertiesPanel from '@/features/template-editor/ui/PropertiesPanel.vue'
import TemplateCanvas from '@/features/template-editor/ui/TemplateCanvas.vue'
import { readData, readError } from '@/shared/api/http'
import AppButton from '@/shared/ui/AppButton.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'

const route = useRoute()
const router = useRouter()
const isNew = computed(() => route.params.id === 'new')
const editor = useTemplateEditor(createDefaultConfig())
const assets = ref<Asset[]>([])
const title = ref('Новый шаблон')
const description = ref('')
const previewAssetId = ref<string | null>(null)
const error = ref('')
const loading = ref(true)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const assetResponse = await fetch('/api/assets')
    assets.value = (await readData<{ assets: Asset[] }>(assetResponse)).assets
    if (!isNew.value) {
      const item = await readData<VideoTemplate>(
        await fetch(`/api/templates/${route.params.id}`),
      )
      title.value = item.name
      description.value = item.description
      previewAssetId.value = item.previewAssetId
      editor.replace(item.config)
    }
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось открыть редактор'
  } finally {
    loading.value = false
  }
}

async function save() {
  error.value = ''
  const missingAsset = editor.draft.value.layers.find(
    (layer) =>
      ['asset_video', 'image', 'gif', 'audio'].includes(layer.type) &&
      !layer.assetId,
  )
  if (missingAsset) {
    error.value = `Для слоя «${missingAsset.name}» выберите ассет в правой панели.`
    return
  }
  saving.value = true
  const response = await fetch(
    isNew.value ? '/api/templates' : `/api/templates/${route.params.id}`,
    {
      method: isNew.value ? 'POST' : 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: title.value,
        description: description.value,
        previewAssetId: previewAssetId.value,
        config: editor.draft.value,
      }),
    },
  )
  if (!response.ok) {
    error.value = await readError(response, 'Не удалось сохранить шаблон')
    saving.value = false
    return
  }
  const saved = await readData<VideoTemplate>(response)
  editor.markSaved()
  saving.value = false
  if (isNew.value) await router.replace(`/templates/${saved.id}`)
}

function beforeUnload(event: BeforeUnloadEvent) {
  if (!editor.isDirty.value) return
  event.preventDefault()
  event.returnValue = ''
}
onMounted(() => {
  void load()
  window.addEventListener('beforeunload', beforeUnload)
})
onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload))
onBeforeRouteLeave(() =>
  editor.isDirty.value
    ? window.confirm('Есть несохранённые изменения. Уйти без сохранения?')
    : true,
)
</script>

<template>
  <section>
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <RouterLink
          to="/templates"
          class="inline-flex items-center gap-1 text-sm text-violet-700"
          ><ArrowLeft class="size-4" />К списку шаблонов</RouterLink
        >
        <h1 class="mt-2 text-xl font-semibold">
          {{ isNew ? 'Новый шаблон' : 'Редактор шаблона' }}
        </h1>
        <p class="mt-1 text-slate-600">
          Перетаскивайте и растягивайте слои прямо на уменьшенном canvas 9:16.
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <AppButton
          variant="secondary"
          :disabled="!editor.canUndo.value"
          @click="editor.undo"
          ><Undo2 class="mr-1 inline size-4" />Назад</AppButton
        >
        <AppButton
          variant="secondary"
          :disabled="!editor.canRedo.value"
          @click="editor.redo"
          ><Redo2 class="mr-1 inline size-4" />Вперёд</AppButton
        >
        <AppButton :disabled="saving" @click="save"
          ><Save class="mr-1 inline size-4" />
          {{ saving ? 'Сохраняем…' : 'Сохранить' }}</AppButton
        >
      </div>
    </div>
    <ErrorState v-if="error" :message="error" />
    <p v-if="editor.isDirty.value" class="mt-3 text-sm text-amber-700">
      Есть несохранённые изменения.
    </p>
    <div v-if="loading" class="mt-8 text-slate-500">Загружаем редактор…</div>
    <div
      v-else
      class="mt-6 grid gap-4 xl:grid-cols-[260px_minmax(380px,1fr)_280px]"
    >
      <LayerPanel
        :layers="editor.draft.value.layers"
        :selected-layer-id="editor.selectedLayerID.value"
        @select="editor.selectedLayerID.value = $event"
        @add="editor.addLayer"
        @update="editor.updateLayer"
        @duplicate="editor.duplicateLayer"
        @remove="editor.removeLayer"
        @move="editor.moveLayer"
      />
      <TemplateCanvas
        :config="editor.draft.value"
        :selected-layer-id="editor.selectedLayerID.value"
        @select="editor.selectedLayerID.value = $event"
        @update="editor.updateLayer"
      />
      <PropertiesPanel
        :layer="editor.selectedLayer.value"
        :assets="assets"
        @update="editor.selectedLayer.value && editor.updateLayer(editor.selectedLayer.value.id, $event)"
      />
    </div>
    <section
      v-if="!loading"
      class="mt-6 rounded-xl border border-slate-200 bg-white p-4"
    >
      <h2 class="font-semibold">Настройки шаблона</h2>
      <div class="mt-3 grid gap-3 md:grid-cols-2">
        <label class="text-sm"
          >Название<input v-model="title" class="mt-1 w-full"></label
        >
        <label class="text-sm"
          >Превью (необязательно)<select
            v-model="previewAssetId"
            class="mt-1 w-full"
          >
            <option :value="null">Без картинки</option>
            <option
              v-for="asset in assets.filter((item) => item.kind === 'image' || item.kind === 'gif')"
              :key="asset.id"
              :value="asset.id"
            >
              {{ asset.name }}
            </option>
          </select></label
        >
        <label class="text-sm md:col-span-2"
          >Описание<textarea
            v-model="description"
            class="mt-1 w-full rounded border border-slate-300 px-3 py-2"
            rows="3"
          /></label
        >
      </div>
    </section>
  </section>
</template>
