<script setup lang="ts">
const nav = [
  { url: '/search', label: 'Поиск клипов' },
  { url: '/pipeline', label: 'Pipeline' },
  { url: '/streamers', label: 'Стримеры' },
  {
    url: 'http://localhost:9001/browser/finde-media',
    label: 's3',
    external: true,
  },
]
</script>

<template>
  <header class="mb-8 flex items-center justify-between">
    <h1 class="text-2xl font-bold">vigun</h1>
    <nav class="flex gap-4">
      <template v-for="item in nav" :key="item.url">
        <RouterLink
          v-if="!item.external"
          :to="item.url"
          class="text-slate-600 hover:text-violet-700"
        >
          {{ item.label }}
        </RouterLink>

        <a
          v-else
          :href="item.url"
          target="_blank"
          rel="noopener noreferrer"
          class="text-slate-600 hover:text-violet-700"
        >
          {{ item.label }}
        </a>
      </template>
    </nav>
  </header>
</template>
