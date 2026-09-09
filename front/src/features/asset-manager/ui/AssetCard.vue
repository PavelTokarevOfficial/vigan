<script setup lang="ts">
import { MoreHorizontal, Pencil, Trash2 } from '@lucide/vue'
import type { Asset } from '@/entities/asset/model/types'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/shared/ui/shadcn/dropdown-menu'

defineProps<{ asset: Asset }>()
defineEmits<{
  preview: [asset: Asset]
  rename: [asset: Asset]
  delete: [asset: Asset]
  dragStart: [event: DragEvent, asset: Asset]
  dragEnd: []
}>()
</script>

<template>
  <article
    draggable="true"
    class="group relative cursor-grab overflow-hidden rounded-xl border border-slate-200 bg-white transition hover:border-violet-300 active:cursor-grabbing"
    @dragstart="$emit('dragStart', $event, asset)"
    @dragend="$emit('dragEnd')"
  >
    <button
      type="button"
      class="block w-full cursor-pointer text-left"
      @click="$emit('preview', asset)"
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
        ><DropdownMenuItem @select="$emit('rename', asset)"
          ><Pencil />Переименовать</DropdownMenuItem
        ><DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive" @select="$emit('delete', asset)"
          ><Trash2 />Удалить</DropdownMenuItem
        ></DropdownMenuContent
      >
    </DropdownMenu>
  </article>
</template>
