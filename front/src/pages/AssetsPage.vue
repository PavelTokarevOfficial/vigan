<script setup lang="ts">
import { Folder, FolderPlus, MoreHorizontal, Pencil, Trash2 } from '@lucide/vue'
import { computed, onMounted, ref } from 'vue'
import type { AssetFolder } from '@/entities/asset/model/types'
import { useAssetManager } from '@/features/asset-manager/model/useAssetManager'
import AssetCard from '@/features/asset-manager/ui/AssetCard.vue'
import AssetManagerDialogs from '@/features/asset-manager/ui/AssetManagerDialogs.vue'
import AssetUploadDropzone from '@/features/asset-manager/ui/AssetUploadDropzone.vue'
import EmptyState from '@/shared/ui/EmptyState.vue'
import ErrorState from '@/shared/ui/ErrorState.vue'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/shared/ui/shadcn/dropdown-menu'

const manager = useAssetManager()
const { assets, busy, error, folders } = manager
const currentFolderID = ref<string | null>(null)
const draggedItem = ref<{ kind: 'asset' | 'folder'; id: string } | null>(null)
const dropTargetID = ref<string | null | undefined>(undefined)

type FolderTreeItem = { folder: AssetFolder; depth: number }

const visibleAssets = computed(() =>
  assets.value.filter((asset) => asset.folderId === currentFolderID.value),
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

function selectFolder(id: string | null) {
  currentFolderID.value = id
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
  if (item.kind === 'asset') {
    const asset = assets.value.find((value) => value.id === item.id)
    if (asset) await manager.moveAsset(asset, parentID)
    return
  }
  const folder = folders.value.find((value) => value.id === item.id)
  if (folder) await manager.moveFolder(folder, parentID)
}
function handleFolderDeleted(folderID: string) {
  if (currentFolderID.value === folderID) currentFolderID.value = null
}

onMounted(() => void manager.load())
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
              ><DropdownMenuItem @select="manager.openCreateFolder(null)"
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
              ><DropdownMenuItem
                @select="manager.openCreateFolder(item.folder.id)"
                ><FolderPlus />Создать дочернюю</DropdownMenuItem
              ><DropdownMenuItem @select="manager.openRenameFolder(item.folder)"
                ><Pencil />Переименовать</DropdownMenuItem
              ><DropdownMenuSeparator />
              <DropdownMenuItem
                variant="destructive"
                @select="manager.openDeleteFolder(item.folder)"
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
        <AssetUploadDropzone
          :busy="busy"
          @files="manager.upload($event, currentFolderID)"
        />
        <EmptyState
          v-if="!visibleFolders.length && !visibleAssets.length"
          class="mt-4"
          message="В этой папке пока нет файлов или вложенных папок."
        />
        <div v-else class="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
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
          <AssetCard
            v-for="asset in visibleAssets"
            :key="asset.id"
            :asset="asset"
            @preview="manager.openAssetPreview"
            @rename="manager.openRenameAsset"
            @delete="manager.openDeleteAsset"
            @drag-start="(event, item) => startDrag(event, 'asset', item.id)"
            @drag-end="endDrag"
          />
        </div>
      </div>
    </div>
    <AssetManagerDialogs
      :manager="manager"
      @folder-deleted="handleFolderDeleted"
    />
  </section>
</template>
