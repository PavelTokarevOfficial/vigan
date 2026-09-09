<script setup lang="ts">
import { Settings } from '@lucide/vue'
import Logo from '@/shared/ui/Logo.vue'

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/shared/ui/shadcn/dropdown-menu'

const primaryNavigation = [
  { url: '/search', label: 'Поиск клипов' },
  { url: '/pipeline', label: 'Pipeline' },
]
const menuNavigation = [
  { url: '/streamers', label: 'Стримеры' },
  {
    url: 'http://localhost:9001/browser/finde-media',
    label: 'S3-хранилище',
    external: true,
  },
]
</script>

<template>
  <header class="mb-8 flex items-center justify-between gap-4">
    <div class="flex items-center gap-6">
      <RouterLink to="/search" class="text-2xl font-bold text-slate-950">
        <Logo class='max-w-[100px] h-auto'/>
      </RouterLink>

      <nav class="flex items-center gap-4">
        <RouterLink
          v-for="item in primaryNavigation"
          :key="item.url"
          :to="item.url"
          class="text-slate-600 hover:text-violet-700"
        >
          {{ item.label }}
        </RouterLink>
      </nav>
    </div>

    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <button
          type="button"
          aria-label="Открыть меню"
          class="rounded-md p-2 text-slate-600 hover:bg-slate-200 hover:text-slate-950 focus:outline-none focus:ring-2 focus:ring-violet-600"
        >
          <Settings class="size-6" aria-hidden="true" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" class="w-48">
        <template v-for="(item, index) in menuNavigation" :key="item.url">
          <DropdownMenuSeparator v-if="index > 0" />
          <DropdownMenuItem as-child>
            <a
              v-if="item.external"
              :href="item.url"
              target="_blank"
              rel="noopener noreferrer"
              class="w-full"
            >
              {{ item.label }}
            </a>
            <RouterLink v-else :to="item.url" class="w-full">
              {{ item.label }}
            </RouterLink>
          </DropdownMenuItem>
        </template>
      </DropdownMenuContent>
    </DropdownMenu>
  </header>
</template>
