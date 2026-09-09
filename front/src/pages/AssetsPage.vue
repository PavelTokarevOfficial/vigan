<script setup lang="ts">
import { FolderPlus, Pencil, Trash2, Upload } from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import type {
  Asset,
  AssetFolder,
  AssetLibrary,
} from '@/entities/asset/model/types'
import { readData, readError } from '@/shared/api/http'
import AppButton from '@/shared/ui/AppButton.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'

const folders = ref<AssetFolder[]>([])
const assets = ref<Asset[]>([])
const currentFolderID = ref<string | null>(null)
const folderName = ref('')
const folderParentID = ref<string | null>(null)
const newFolderName = ref('')
const files = ref<FileList | null>(null)
const error = ref('')
const busy = ref(false)

const visibleAssets = computed(() =>
  currentFolderID.value
    ? assets.value.filter((asset) => asset.folderId === currentFolderID.value)
    : assets.value.filter((asset) => asset.folderId === null),
)
const currentFolder = computed(
  () =>
    folders.value.find((folder) => folder.id === currentFolderID.value) ?? null,
)

async function load() {
  try {
    const library = await readData<AssetLibrary>(await fetch('/api/assets'))
    folders.value = library.folders
    assets.value = library.assets
  } catch (cause) {
    error.value =
      cause instanceof Error ? cause.message : 'Не удалось загрузить ассеты'
  }
}
async function createFolder() {
  if (!newFolderName.value.trim()) return
  busy.value = true
  const response = await fetch('/api/asset-folders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: newFolderName.value,
      parentId: currentFolderID.value,
    }),
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось создать папку')
  else newFolderName.value = ''
  busy.value = false
  await load()
}
async function renameFolder() {
  if (!currentFolder.value || !folderName.value.trim()) return
  busy.value = true
  const response = await fetch(`/api/asset-folders/${currentFolder.value.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: folderName.value,
      parentId: folderParentID.value,
    }),
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось переименовать папку')
  busy.value = false
  await load()
}
async function removeFolder() {
  if (
    !currentFolder.value ||
    !window.confirm(`Удалить папку «${currentFolder.value.name}»?`)
  )
    return
  busy.value = true
  const response = await fetch(`/api/asset-folders/${currentFolder.value.id}`, {
    method: 'DELETE',
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось удалить папку')
  else {
    currentFolderID.value = null
    folderName.value = ''
  }
  busy.value = false
  await load()
}
async function upload() {
  if (!files.value?.length) return
  busy.value = true
  error.value = ''
  for (const file of Array.from(files.value)) {
    const form = new FormData()
    form.append('file', file)
    if (currentFolderID.value) form.append('folderId', currentFolderID.value)
    const response = await fetch('/api/assets', { method: 'POST', body: form })
    if (!response.ok) {
      error.value = await readError(
        response,
        `Не удалось загрузить ${file.name}`,
      )
      break
    }
  }
  files.value = null
  busy.value = false
  await load()
}
async function renameAsset(asset: Asset) {
  const name = window.prompt('Новое имя файла', asset.name)
  if (!name?.trim()) return
  const response = await fetch(`/api/assets/${asset.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, folderId: asset.folderId }),
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось переименовать ассет')
  await load()
}
async function moveAsset(asset: Asset, folderID: string | null) {
  const response = await fetch(`/api/assets/${asset.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: asset.name, folderId: folderID || null }),
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось переместить ассет')
  await load()
}
async function removeAsset(asset: Asset) {
  if (!window.confirm(`Удалить «${asset.name}» из хранилища?`)) return
  const response = await fetch(`/api/assets/${asset.id}`, { method: 'DELETE' })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось удалить ассет')
  await load()
}
function selectFolder(id: string | null) {
  currentFolderID.value = id
  const folder = folders.value.find((item) => item.id === id)
  folderName.value = folder?.name || ''
  folderParentID.value = folder?.parentId || null
}

onMounted(() => void load())
</script>

<template>
  <section>
    <h1 class="text-xl font-semibold">Ассеты</h1>
    <p class="mt-1 text-slate-600">
      Видео, GIF, изображения и аудио в S3. Файлы не хранятся во фронтенде или
      базе.
    </p>
    <ErrorState v-if="error" :message="error" />
    <div class="mt-6 grid gap-4 lg:grid-cols-[240px_minmax(0,1fr)]">
      <aside class="rounded-xl border border-slate-200 bg-white p-3">
        <button
          type="button"
          class="w-full rounded px-2 py-1 text-left text-sm hover:bg-slate-100"
          :class="currentFolderID === null ? 'bg-violet-50 text-violet-800' : ''"
          @click="selectFolder(null)"
        >
          Корень
        </button>
        <button
          v-for="folder in folders"
          :key="folder.id"
          type="button"
          class="mt-1 w-full truncate rounded px-2 py-1 text-left text-sm hover:bg-slate-100"
          :class="currentFolderID === folder.id ? 'bg-violet-50 text-violet-800' : ''"
          @click="selectFolder(folder.id)"
        >
          📁 {{ folder.name }}
        </button>
        <div class="mt-4 border-t pt-3">
          <label class="text-sm"
            >Новая папка<input
              v-model="newFolderName"
              class="mt-1 w-full"
              placeholder="Например, интро"
            ></label
          >
          <AppButton
            class="mt-2 w-full"
            variant="secondary"
            :disabled="busy"
            @click="createFolder"
            ><FolderPlus class="mr-1 inline size-4" />Создать</AppButton
          >
        </div>
      </aside>
      <div>
        <div
          class="flex flex-wrap items-end justify-between gap-3 rounded-xl border border-slate-200 bg-white p-4"
        >
          <div>
            <h2 class="font-semibold">{{ currentFolder?.name || 'Корень' }}</h2>
            <div v-if="currentFolder" class="mt-2 flex flex-wrap gap-2">
              <input
                v-model="folderName"
                class="max-w-56"
                aria-label="Название текущей папки"
              ><select
                v-model="folderParentID"
                class="max-w-56"
                aria-label="Родительская папка"
              >
                <option :value="null">Корень</option>
                <option
                  v-for="folder in folders.filter((item) => item.id !== currentFolder?.id)"
                  :key="folder.id"
                  :value="folder.id"
                >
                  {{ folder.name }}
                </option>
              </select><AppButton
                variant="secondary"
                :disabled="busy"
                @click="renameFolder"
                ><Pencil class="mr-1 inline size-4" />Сохранить папку</AppButton
              ><AppButton
                variant="danger"
                :disabled="busy"
                @click="removeFolder"
                ><Trash2 class="mr-1 inline size-4" />Удалить папку</AppButton
              >
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <input
              type="file"
              multiple
              accept="image/*,video/*,audio/*"
              @change="files = ($event.target as HTMLInputElement).files"
            ><AppButton :disabled="busy || !files?.length" @click="upload"
              ><Upload class="mr-1 inline size-4" />Загрузить</AppButton
            >
          </div>
        </div>
        <EmptyState
          v-if="!visibleAssets.length"
          class="mt-4"
          message="В этой папке пока нет ассетов."
        />
        <div v-else class="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <article
            v-for="asset in visibleAssets"
            :key="asset.id"
            class="overflow-hidden rounded-xl border border-slate-200 bg-white"
          >
            <video
              v-if="asset.kind === 'video'"
              :src="asset.url"
              controls
              class="aspect-video w-full bg-black object-contain"
            />
            <audio
              v-else-if="asset.kind === 'audio'"
              :src="asset.url"
              controls
              class="w-full p-4"
            />
            <img
              v-else
              :src="asset.url"
              :alt="asset.name"
              class="aspect-video w-full object-contain bg-slate-100"
            >
            <div class="p-3">
              <p class="truncate font-medium">{{ asset.name }}</p>
              <p class="mt-1 text-xs text-slate-500">
                {{ asset.kind }} · {{ Math.round(asset.size / 1024) }} KB
              </p>
              <div class="mt-3 flex flex-wrap gap-2">
                <AppButton variant="secondary" @click="renameAsset(asset)"
                  >Переименовать</AppButton
                ><select
                  :value="asset.folderId"
                  aria-label="Папка ассета"
                  @change="moveAsset(asset, ($event.target as HTMLSelectElement).value || null)"
                >
                  <option value="">Корень</option>
                  <option
                    v-for="folder in folders"
                    :key="folder.id"
                    :value="folder.id"
                  >
                    {{ folder.name }}
                  </option>
                </select><AppButton variant="danger" @click="removeAsset(asset)"
                  >Удалить</AppButton
                >
              </div>
            </div>
          </article>
        </div>
      </div>
    </div>
  </section>
</template>
