import { ref } from 'vue'
import type {
  Asset,
  AssetFolder,
  AssetLibrary,
} from '@/entities/asset/model/types'
import { readData, readError } from '@/shared/api/http'

export function useAssetManager() {
  const folders = ref<AssetFolder[]>([])
  const assets = ref<Asset[]>([])
  const error = ref('')
  const busy = ref(false)

  const createFolderOpen = ref(false)
  const createFolderParentID = ref<string | null>(null)
  const newFolderName = ref('')
  const renameFolderOpen = ref(false)
  const renameFolderTarget = ref<AssetFolder | null>(null)
  const renameFolderName = ref('')
  const deleteFolderTarget = ref<AssetFolder | null>(null)
  const previewAsset = ref<Asset | null>(null)
  const renameAssetTarget = ref<Asset | null>(null)
  const renameAssetName = ref('')
  const deleteAssetTarget = ref<Asset | null>(null)

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

  async function request(
    source: Response | Promise<Response>,
    fallback: string,
  ) {
    const response = await source
    if (!response.ok) throw new Error(await readError(response, fallback))
  }
  async function run(action: () => Promise<void>) {
    if (busy.value) return false
    busy.value = true
    error.value = ''
    try {
      await action()
      await load()
      return true
    } catch (cause) {
      error.value =
        cause instanceof Error ? cause.message : 'Неизвестная ошибка'
      return false
    } finally {
      busy.value = false
    }
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
  async function createFolder() {
    if (!newFolderName.value.trim()) return false
    const created = await run(() =>
      request(
        fetch('/api/asset-folders', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: newFolderName.value,
            parentId: createFolderParentID.value,
          }),
        }),
        'Не удалось создать папку',
      ),
    )
    if (created) closeCreateFolder()
    return created
  }

  function openRenameFolder(folder: AssetFolder) {
    renameFolderTarget.value = folder
    renameFolderName.value = folder.name
    renameFolderOpen.value = true
  }
  function closeRenameFolder() {
    renameFolderTarget.value = null
    renameFolderName.value = ''
    renameFolderOpen.value = false
  }
  async function renameFolder() {
    const folder = renameFolderTarget.value
    if (!folder || !renameFolderName.value.trim()) return false
    const renamed = await run(() =>
      request(
        fetch(`/api/asset-folders/${folder.id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: renameFolderName.value,
            parentId: folder.parentId,
          }),
        }),
        'Не удалось переименовать папку',
      ),
    )
    if (renamed) closeRenameFolder()
    return renamed
  }

  function openDeleteFolder(folder: AssetFolder) {
    deleteFolderTarget.value = folder
  }
  function closeDeleteFolder() {
    deleteFolderTarget.value = null
  }
  async function removeFolder() {
    const folder = deleteFolderTarget.value
    if (!folder) return false
    const removed = await run(() =>
      request(
        fetch(`/api/asset-folders/${folder.id}`, { method: 'DELETE' }),
        'Не удалось удалить папку',
      ),
    )
    if (removed) closeDeleteFolder()
    return removed
  }

  async function upload(files: FileList, folderID: string | null) {
    if (!files.length || busy.value) return false
    return run(async () => {
      for (const file of Array.from(files)) {
        const form = new FormData()
        form.append('file', file)
        if (folderID) form.append('folderId', folderID)
        await request(
          fetch('/api/assets', { method: 'POST', body: form }),
          `Не удалось загрузить ${file.name}`,
        )
      }
    })
  }

  async function moveAsset(asset: Asset, folderID: string | null) {
    if (asset.folderId === folderID) return true
    return run(() =>
      request(
        fetch(`/api/assets/${asset.id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: asset.name, folderId: folderID }),
        }),
        'Не удалось переместить ассет',
      ),
    )
  }
  async function moveFolder(folder: AssetFolder, parentID: string | null) {
    if (folder.id === parentID || folder.parentId === parentID) return true
    return run(() =>
      request(
        fetch(`/api/asset-folders/${folder.id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name: folder.name, parentId: parentID }),
        }),
        'Не удалось переместить папку',
      ),
    )
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
    if (!asset || !renameAssetName.value.trim()) return false
    const renamed = await run(() =>
      request(
        fetch(`/api/assets/${asset.id}`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: renameAssetName.value,
            folderId: asset.folderId,
          }),
        }),
        'Не удалось переименовать ассет',
      ),
    )
    if (renamed) closeRenameAsset()
    return renamed
  }
  function openDeleteAsset(asset: Asset) {
    deleteAssetTarget.value = asset
  }
  function closeDeleteAsset() {
    deleteAssetTarget.value = null
  }
  async function removeAsset() {
    const asset = deleteAssetTarget.value
    if (!asset) return false
    const removed = await run(() =>
      request(
        fetch(`/api/assets/${asset.id}`, { method: 'DELETE' }),
        'Не удалось удалить ассет',
      ),
    )
    if (removed) {
      if (previewAsset.value?.id === asset.id) closeAssetPreview()
      closeDeleteAsset()
    }
    return removed
  }

  return {
    folders,
    assets,
    error,
    busy,
    createFolderOpen,
    createFolderParentID,
    newFolderName,
    renameFolderOpen,
    renameFolderTarget,
    renameFolderName,
    deleteFolderTarget,
    previewAsset,
    renameAssetTarget,
    renameAssetName,
    deleteAssetTarget,
    load,
    openCreateFolder,
    closeCreateFolder,
    createFolder,
    openRenameFolder,
    closeRenameFolder,
    renameFolder,
    openDeleteFolder,
    closeDeleteFolder,
    removeFolder,
    upload,
    moveAsset,
    moveFolder,
    openAssetPreview,
    closeAssetPreview,
    openRenameAsset,
    closeRenameAsset,
    renameAsset,
    openDeleteAsset,
    closeDeleteAsset,
    removeAsset,
  }
}

export type AssetManager = ReturnType<typeof useAssetManager>
