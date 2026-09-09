<script setup lang="ts">
const nav = [
  ['/streamers', 'Стримеры'],
  ['/clips', 'Клипы'],
  ['/pipeline', 'Pipeline'],
  ['/videos', 'Видео'],
]
</script>

<template>
  <header class="mb-8 flex items-center justify-between">
    <h1 class="text-2xl font-bold">Finde Clip</h1>
    <nav class="flex gap-4">
      <RouterLink
        v-for="item in nav"
        :key="item[0]"
        :to="item[0]"
        class="text-slate-600 hover:text-violet-700"
        >{{
          item[1]
        }}</RouterLink
      >
    </nav>
  </header>
</template>
