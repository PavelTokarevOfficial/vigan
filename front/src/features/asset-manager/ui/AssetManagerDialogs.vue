<script setup lang="ts">
import type { AssetManager } from '@/features/asset-manager/model/useAssetManager'
import AppButton from '@/shared/ui/AppButton.vue'

const props = defineProps<{ manager: AssetManager }>()
const emit = defineEmits<{ folderDeleted: [folderID: string] }>()
const {
  folders,
  busy,
  createFolderOpen,
  createFolderParentID,
  newFolderName,
  renameFolderOpen,
  renameFolderName,
  deleteFolderTarget,
  previewAsset,
  renameAssetTarget,
  renameAssetName,
  deleteAssetTarget,
} = props.manager

async function removeFolder() {
  const folderID = deleteFolderTarget.value?.id
  if ((await props.manager.removeFolder()) && folderID)
    emit('folderDeleted', folderID)
}
</script>

<template>
  <div
    v-if="createFolderOpen"
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby="create-folder-title"
  >
    <form
      class="w-full max-w-sm rounded-xl bg-white p-5 shadow-2xl"
      @submit.prevent="manager.createFolder"
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
        <AppButton variant="secondary" @click="manager.closeCreateFolder"
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
      @submit.prevent="manager.renameFolder"
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
        <AppButton variant="secondary" @click="manager.closeRenameFolder"
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
        Папка «{{ deleteFolderTarget.name }}» будет удалена. Сначала переместите
        или удалите вложенные папки и файлы.
      </p>
      <div class="mt-5 flex justify-end gap-2">
        <AppButton variant="secondary" @click="manager.closeDeleteFolder"
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
        <AppButton variant="secondary" @click="manager.closeAssetPreview"
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
      @submit.prevent="manager.renameAsset"
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
        <AppButton variant="secondary" @click="manager.closeRenameAsset"
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
        <AppButton variant="secondary" @click="manager.closeDeleteAsset"
          >Отмена</AppButton
        ><AppButton
          variant="danger"
          :disabled="busy"
          @click="manager.removeAsset"
          >Удалить</AppButton
        >
      </div>
    </section>
  </div>
</template>
