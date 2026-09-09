<script setup lang="ts">
import {
  Folder,
  FolderPlus,
  MoreHorizontal,
  Pencil,
  Trash2,
  Upload,
} from '@lucide/vue'
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/shared/ui/shadcn/dropdown-menu'

const folders = ref<AssetFolder[]>([])
const assets = ref<Asset[]>([])
const currentFolderID = ref<string | null>(null)
const newFolderName = ref('')
const createFolderParentID = ref<string | null>(null)
const createFolderOpen = ref(false)
const renameFolderOpen = ref(false)
const renameFolderID = ref<string | null>(null)
const renameFolderName = ref('')
const deleteFolderTarget = ref<AssetFolder | null>(null)
const uploadDropActive = ref(false)
const previewAsset = ref<Asset | null>(null)
const renameAssetTarget = ref<Asset | null>(null)
const renameAssetName = ref('')
const deleteAssetTarget = ref<Asset | null>(null)
const error = ref('')
const busy = ref(false)
const draggedItem = ref<{ kind: 'asset' | 'folder'; id: string } | null>(null)
const dropTargetID = ref<string | null | undefined>(undefined)

type FolderTreeItem = { folder: AssetFolder; depth: number }

const visibleAssets = computed(() =>
  currentFolderID.value
    ? assets.value.filter((asset) => asset.folderId === currentFolderID.value)
    : assets.value.filter((asset) => asset.folderId === null),
)
const visibleFolders = computed(() =>
  folders.value
    .filter((folder) => (folder.parentId ?? null) === currentFolderID.value)
    .sort((left, right) => left.name.localeCompare(right.name, 'ru')),
)
const currentFolder = computed(
  () =>
    folders.value.find((folder) => folder.id === currentFolderID.value) ?? null,
)
const folderTree = computed<FolderTreeItem[]>(() => {
  const children = new Map<string | null, AssetFolder[]>()
  for (const folder of folders.value) {
    const parentID =
      folder.parentId &&
      folders.value.some((item) => item.id === folder.parentId)
        ? folder.parentId
        : null
    children.set(parentID, [...(children.get(parentID) ?? []), folder])
  }
  for (const items of children.values())
    items.sort((left, right) => left.name.localeCompare(right.name, 'ru'))

  const result: FolderTreeItem[] = []
  const visited = new Set<string>()
  function addBranch(parentID: string | null, depth: number) {
    for (const folder of children.get(parentID) ?? []) {
      if (visited.has(folder.id)) continue
      visited.add(folder.id)
      result.push({ folder, depth })
      addBranch(folder.id, depth + 1)
    }
  }
  addBranch(null, 0)
  return result
})

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
      parentId: createFolderParentID.value,
    }),
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось создать папку')
  else {
    newFolderName.value = ''
    createFolderParentID.value = null
    createFolderOpen.value = false
  }
  busy.value = false
  await load()
}
function openCreateFolder(parentID: string | null) {
  createFolderParentID.value = parentID
  newFolderName.value = ''
  createFolderOpen.value = true
}
function closeCreateFolder() {
  createFolderParentID.value = null
  newFolderName.value = ''
  createFolderOpen.value = false
}
function openRenameFolder(folder: AssetFolder) {
  renameFolderID.value = folder.id
  renameFolderName.value = folder.name
  renameFolderOpen.value = true
}
function closeRenameFolder() {
  renameFolderID.value = null
  renameFolderName.value = ''
  renameFolderOpen.value = false
}
async function renameFolder() {
  const folder = folders.value.find((item) => item.id === renameFolderID.value)
  if (!folder || !renameFolderName.value.trim()) return
  busy.value = true
  const response = await fetch(`/api/asset-folders/${folder.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: renameFolderName.value,
      parentId: folder.parentId,
    }),
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось переименовать папку')
  else closeRenameFolder()
  busy.value = false
  await load()
}
function openDeleteFolder(folder: AssetFolder) {
  deleteFolderTarget.value = folder
}
function closeDeleteFolder() {
  deleteFolderTarget.value = null
}
async function removeFolder() {
  const folder = deleteFolderTarget.value
  if (!folder) return
  busy.value = true
  const response = await fetch(`/api/asset-folders/${folder.id}`, {
    method: 'DELETE',
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось удалить папку')
  else {
    if (currentFolderID.value === folder.id) currentFolderID.value = null
    closeDeleteFolder()
  }
  busy.value = false
  await load()
}
async function upload(filesToUpload: FileList) {
  if (!filesToUpload.length || busy.value) return
  busy.value = true
  error.value = ''
  for (const file of Array.from(filesToUpload)) {
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
  busy.value = false
  await load()
}
function selectFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const selectedFiles = input.files
  if (selectedFiles) void upload(selectedFiles)
  input.value = ''
}
function dropFiles(event: DragEvent) {
  uploadDropActive.value = false
  const selectedFiles = event.dataTransfer?.files
  if (selectedFiles?.length) void upload(selectedFiles)
}
function openAssetPreview(asset: Asset) {
  previewAsset.value = asset
}
function closeAssetPreview() {
  previewAsset.value = null
}
function openRenameAsset(asset: Asset) {
  renameAssetTarget.value = asset
  renameAssetName.value = asset.name
}
function closeRenameAsset() {
  renameAssetTarget.value = null
  renameAssetName.value = ''
}
async function renameAsset() {
  const asset = renameAssetTarget.value
  if (!asset || !renameAssetName.value.trim()) return
  busy.value = true
  const response = await fetch(`/api/assets/${asset.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: renameAssetName.value,
      folderId: asset.folderId,
    }),
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось переименовать ассет')
  else closeRenameAsset()
  busy.value = false
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
async function moveFolder(folder: AssetFolder, parentID: string | null) {
  const response = await fetch(`/api/asset-folders/${folder.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: folder.name, parentId: parentID }),
  })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось переместить папку')
  await load()
}
function startDrag(event: DragEvent, kind: 'asset' | 'folder', id: string) {
  draggedItem.value = { kind, id }
  event.dataTransfer?.setData('text/plain', `${kind}:${id}`)
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move'
}
function endDrag() {
  draggedItem.value = null
  dropTargetID.value = undefined
}
function setDropTarget(id: string | null) {
  if (draggedItem.value) dropTargetID.value = id
}
function clearDropTarget(id: string | null) {
  if (dropTargetID.value === id) dropTargetID.value = undefined
}
function isDropTarget(id: string | null) {
  return draggedItem.value !== null && dropTargetID.value === id
}
async function dropIntoFolder(parentID: string | null) {
  const item = draggedItem.value
  endDrag()
  if (!item || busy.value) return
  error.value = ''
  busy.value = true
  try {
    if (item.kind === 'asset') {
      const asset = assets.value.find((value) => value.id === item.id)
      if (!asset || asset.folderId === parentID) return
      await moveAsset(asset, parentID)
      return
    }
    const folder = folders.value.find((value) => value.id === item.id)
    if (!folder || folder.id === parentID || folder.parentId === parentID)
      return
    await moveFolder(folder, parentID)
  } finally {
    busy.value = false
  }
}
function openDeleteAsset(asset: Asset) {
  deleteAssetTarget.value = asset
}
function closeDeleteAsset() {
  deleteAssetTarget.value = null
}
async function removeAsset() {
  const asset = deleteAssetTarget.value
  if (!asset) return
  busy.value = true
  const response = await fetch(`/api/assets/${asset.id}`, { method: 'DELETE' })
  if (!response.ok)
    error.value = await readError(response, 'Не удалось удалить ассет')
  else {
    if (previewAsset.value?.id === asset.id) closeAssetPreview()
    closeDeleteAsset()
  }
  busy.value = false
  await load()
}
function selectFolder(id: string | null) {
  currentFolderID.value = id
}

onMounted(() => void load())
</script>

<template>
  <section>
    <h1 class="text-4xl font-semibold mb-10">Ассеты</h1>
    
    <ErrorState v-if="error" :message="error" />
    <div class="grid gap-4 lg:grid-cols-[240px_minmax(0,1fr)]">
      <aside class="rounded-xl border border-slate-200 bg-white p-3">
        <!-- biome-ignore lint/a11y/noStaticElementInteractions: native drop target around the root tree item -->
        <div
          class="group flex w-full items-center rounded px-2 py-1 text-sm hover:bg-slate-100"
          :class="[
            currentFolderID === null ? 'bg-violet-50 text-violet-800' : '',
            isDropTarget(null) ? 'ring-2 ring-violet-500' : '',
          ]"
          @dragover.prevent="setDropTarget(null)"
          @dragleave="clearDropTarget(null)"
          @drop.prevent="dropIntoFolder(null)"
        >
          <button
            type="button"
            class="min-w-0 flex-1 text-left"
            @click="selectFolder(null)"
          >
            Корень
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger as-child
              ><button
                type="button"
                class="ml-auto rounded p-1 text-slate-600 invisible group-hover:visible hover:bg-slate-200"
                aria-label="Действия с корнем"
              >
                <MoreHorizontal class="size-4" />
              </button></DropdownMenuTrigger
            >
            <DropdownMenuContent align="end" class="w-52"
              ><DropdownMenuItem @select="openCreateFolder(null)"
                ><FolderPlus />Создать папку</DropdownMenuItem
              ></DropdownMenuContent
            >
          </DropdownMenu>
        </div>
        <!-- biome-ignore lint/a11y/noStaticElementInteractions: native drop target around a folder tree item -->
        <div
          v-for="item in folderTree"
          :key="item.folder.id"
          draggable="true"
          class="group mt-1 flex w-full min-w-0 items-center gap-1 truncate rounded py-1 pr-2 text-left text-sm hover:bg-slate-100"
          :class="[
            currentFolderID === item.folder.id ? 'bg-violet-50 text-violet-800' : '',
            isDropTarget(item.folder.id) ? 'ring-2 ring-violet-500' : '',
          ]"
          :style="{ paddingLeft: `${24 + item.depth * 16}px` }"
          @dragstart="startDrag($event, 'folder', item.folder.id)"
          @dragend="endDrag"
          @dragover.prevent="setDropTarget(item.folder.id)"
          @dragleave="clearDropTarget(item.folder.id)"
          @drop.prevent="dropIntoFolder(item.folder.id)"
        >
          <button
            type="button"
            class="flex min-w-0 flex-1 items-center gap-1 text-left"
            @click="selectFolder(item.folder.id)"
          >
            <span class="shrink-0">📁</span
            ><span class="truncate">{{ item.folder.name }}</span>
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger as-child
              ><button
                type="button"
                class="ml-auto rounded p-1 text-slate-600 invisible group-hover:visible hover:bg-slate-200"
                :aria-label="`Действия с папкой ${item.folder.name}`"
              >
                <MoreHorizontal class="size-4" />
              </button></DropdownMenuTrigger
            >
            <DropdownMenuContent align="end" class="w-52"
              ><DropdownMenuItem @select="openCreateFolder(item.folder.id)"
                ><FolderPlus />Создать дочернюю</DropdownMenuItem
              ><DropdownMenuItem @select="openRenameFolder(item.folder)"
                ><Pencil />Переименовать</DropdownMenuItem
              ><DropdownMenuSeparator />
              <DropdownMenuItem
                variant="destructive"
                @select="openDeleteFolder(item.folder)"
                ><Trash2 />Удалить</DropdownMenuItem
              ></DropdownMenuContent
            >
          </DropdownMenu>
        </div>
      </aside>
      <div>
        <h2 class="text-2xl font-semibold">
          {{ currentFolder?.name || 'Корень' }}
        </h2>
        <label
          class="mt-4 flex min-h-48 w-full cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed p-6 text-center transition"
          :class="
            uploadDropActive
              ? 'border-violet-500 bg-violet-50 text-violet-800'
              : 'border-slate-300 bg-white text-slate-600 hover:border-violet-400 hover:bg-violet-50/50'
          "
          @dragenter.prevent="uploadDropActive = true"
          @dragover.prevent="uploadDropActive = true"
          @dragleave="uploadDropActive = false"
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
          <span class="mt-3 font-medium"
            >Нажмите на блок или перетащите файлы</span
          >
          <span class="mt-1 text-sm text-slate-500">
            Изображения, GIF, видео и аудио
          </span>
          <span v-if="busy" class="mt-3 text-sm font-medium"
            >Загрузка файлов…</span
          >
        </label>
        <EmptyState
          v-if="!visibleFolders.length && !visibleAssets.length"
          class="mt-4"
          message="В этой папке пока нет файлов или вложенных папок."
        />
        <div v-else class="mt-4 grid gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 xl:grid-cols-7">
          <button
            v-for="folder in visibleFolders"
            :key="folder.id"
            type="button"
            draggable="true"
            class="flex min-h-36 cursor-pointer flex-col items-start justify-center rounded-xl border border-slate-200 bg-white p-4 text-left transition hover:border-violet-300 hover:bg-violet-50/50"
            @click="selectFolder(folder.id)"
            @dragstart="startDrag($event, 'folder', folder.id)"
            @dragend="endDrag"
          >
            <Folder class="size-8 text-violet-600" />
            <span class="mt-3 w-full truncate font-medium">{{
              folder.name
            }}</span>
            <span class="mt-1 text-sm text-slate-500">Папка</span>
          </button>
          <article
            v-for="asset in visibleAssets"
            :key="asset.id"
            draggable="true"
            class="group relative cursor-grab overflow-hidden rounded-xl border border-slate-200 bg-white transition hover:border-violet-300 active:cursor-grabbing"
            @dragstart="startDrag($event, 'asset', asset.id)"
            @dragend="endDrag"
          >
            <button
              type="button"
              class="block w-full cursor-pointer text-left"
              @click="openAssetPreview(asset)"
            >
              <video
                v-if="asset.kind === 'video'"
                :src="asset.url"
                muted
                preload="metadata"
                class="aspect-video w-full bg-black object-contain"
              />
              <div
                v-else-if="asset.kind === 'audio'"
                class="flex aspect-video items-center justify-center bg-slate-100 text-sm text-slate-600"
              >
                Аудиофайл
              </div>
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
              </div>
            </button>
            <DropdownMenu>
              <DropdownMenuTrigger as-child
                ><button
                  type="button"
                  class="absolute top-2 right-2 rounded-md border border-slate-200 bg-white p-1.5 text-slate-600 shadow-sm opacity-0 transition group-hover:opacity-100 hover:bg-slate-100 focus:opacity-100"
                  :aria-label="`Действия с файлом ${asset.name}`"
                  @click.stop
                >
                  <MoreHorizontal class="size-4" />
                </button></DropdownMenuTrigger
              >
              <DropdownMenuContent align="end" class="w-52"
                ><DropdownMenuItem @select="openRenameAsset(asset)"
                  ><Pencil />Переименовать</DropdownMenuItem
                ><DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="destructive"
                  @select="openDeleteAsset(asset)"
                  ><Trash2 />Удалить</DropdownMenuItem
                ></DropdownMenuContent
              >
            </DropdownMenu>
          </article>
        </div>
      </div>
    </div>
    <div
      v-if="createFolderOpen"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="create-folder-title"
    >
      <form
        class="w-full max-w-sm rounded-xl bg-white p-5 shadow-2xl"
        @submit.prevent="createFolder"
      >
        <h2 id="create-folder-title" class="text-lg font-semibold">
          Новая папка
        </h2>
        <p class="mt-1 text-sm text-slate-600">
          {{
            createFolderParentID
              ? `Внутри «${folders.find((folder) => folder.id === createFolderParentID)?.name}»`
              : 'В корне'
          }}
        </p>
        <label class="mt-4 block text-sm"
          >Название<input
            v-model="newFolderName"
            class="mt-1 w-full"
            placeholder="Например, интро"
          ></label
        >
        <div class="mt-5 flex justify-end gap-2">
          <AppButton variant="secondary" @click="closeCreateFolder"
            >Отмена</AppButton
          ><AppButton type="submit" :disabled="busy || !newFolderName.trim()"
            >Создать</AppButton
          >
        </div>
      </form>
    </div>
    <div
      v-if="renameFolderOpen"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="rename-folder-title"
    >
      <form
        class="w-full max-w-sm rounded-xl bg-white p-5 shadow-2xl"
        @submit.prevent="renameFolder"
      >
        <h2 id="rename-folder-title" class="text-lg font-semibold">
          Переименовать папку
        </h2>
        <label class="mt-4 block text-sm"
          >Новое название<input
            v-model="renameFolderName"
            class="mt-1 w-full"
          ></label
        >
        <div class="mt-5 flex justify-end gap-2">
          <AppButton variant="secondary" @click="closeRenameFolder"
            >Отмена</AppButton
          ><AppButton type="submit" :disabled="busy || !renameFolderName.trim()"
            >Сохранить</AppButton
          >
        </div>
      </form>
    </div>
    <div
      v-if="deleteFolderTarget"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="delete-folder-title"
    >
      <section class="w-full max-w-sm rounded-xl bg-white p-5 shadow-2xl">
        <h2 id="delete-folder-title" class="text-lg font-semibold">
          Удалить папку?
        </h2>
        <p class="mt-2 text-sm text-slate-600">
          Папка «{{ deleteFolderTarget.name }}» будет удалена. Сначала
          переместите или удалите вложенные папки и файлы.
        </p>
        <div class="mt-5 flex justify-end gap-2">
          <AppButton variant="secondary" @click="closeDeleteFolder"
            >Отмена</AppButton
          ><AppButton variant="danger" :disabled="busy" @click="removeFolder"
            >Удалить</AppButton
          >
        </div>
      </section>
    </div>
    <div
      v-if="previewAsset"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="asset-preview-title"
    >
      <section class="w-full max-w-4xl rounded-xl bg-white p-5 shadow-2xl">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <h2 id="asset-preview-title" class="truncate text-lg font-semibold">
              {{ previewAsset.name }}
            </h2>
            <p class="mt-1 text-sm text-slate-500">
              {{ previewAsset.kind }} ·
              {{ Math.round(previewAsset.size / 1024) }} KB
            </p>
          </div>
          <AppButton variant="secondary" @click="closeAssetPreview"
            >Закрыть</AppButton
          >
        </div>
        <div
          class="mt-4 flex min-h-64 items-center justify-center rounded-lg bg-slate-950 p-3"
        >
          <video
            v-if="previewAsset.kind === 'video'"
            :src="previewAsset.url"
            controls
            autoplay
            class="max-h-[70vh] max-w-full"
          />
          <audio
            v-else-if="previewAsset.kind === 'audio'"
            :src="previewAsset.url"
            controls
            autoplay
            class="w-full max-w-xl"
          />
          <img
            v-else
            :src="previewAsset.url"
            :alt="previewAsset.name"
            class="max-h-[70vh] max-w-full object-contain"
          >
        </div>
      </section>
    </div>
    <div
      v-if="renameAssetTarget"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="rename-asset-title"
    >
      <form
        class="w-full max-w-sm rounded-xl bg-white p-5 shadow-2xl"
        @submit.prevent="renameAsset"
      >
        <h2 id="rename-asset-title" class="text-lg font-semibold">
          Переименовать файл
        </h2>
        <label class="mt-4 block text-sm"
          >Новое название<input
            v-model="renameAssetName"
            class="mt-1 w-full"
          ></label
        >
        <div class="mt-5 flex justify-end gap-2">
          <AppButton variant="secondary" @click="closeRenameAsset"
            >Отмена</AppButton
          ><AppButton type="submit" :disabled="busy || !renameAssetName.trim()"
            >Сохранить</AppButton
          >
        </div>
      </form>
    </div>
    <div
      v-if="deleteAssetTarget"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="delete-asset-title"
    >
      <section class="w-full max-w-sm rounded-xl bg-white p-5 shadow-2xl">
        <h2 id="delete-asset-title" class="text-lg font-semibold">
          Удалить файл?
        </h2>
        <p class="mt-2 text-sm text-slate-600">
          Файл «{{ deleteAssetTarget.name }}» будет удалён из хранилища.
        </p>
        <div class="mt-5 flex justify-end gap-2">
          <AppButton variant="secondary" @click="closeDeleteAsset"
            >Отмена</AppButton
          ><AppButton variant="danger" :disabled="busy" @click="removeAsset"
            >Удалить</AppButton
          >
        </div>
      </section>
    </div>
  </section>
</template>
